package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/semmidev/phylax/internal/adapter/compressor"
	"github.com/semmidev/phylax/internal/adapter/database"
	"github.com/semmidev/phylax/internal/adapter/storage"
	"github.com/semmidev/phylax/internal/config"
	"github.com/semmidev/phylax/internal/domain"
	"github.com/semmidev/phylax/internal/infrastructure/logger"
	"github.com/semmidev/phylax/internal/infrastructure/scheduler"
	"github.com/semmidev/phylax/internal/usecase"
)

// App represents the main application.
type App struct {
	config        *config.Config
	logger        *logger.Logger
	scheduler     *scheduler.Scheduler
	uploadTargets []usecase.UploadTarget
	backupJobs    []domain.BackupJob
	cleanupUC     *usecase.Cleanup
	oauthService  OAuthService
}

// New creates a new App instance.
func New(ctx context.Context, cfg *config.Config) (*App, error) {
	if cfg == nil {
		return nil, errors.New("config cannot be nil")
	}

	// Initialize logger
	log, err := logger.New(cfg.App.LogLevel, cfg.App.LogFile)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	log.Infof("Starting %s", cfg.App.Name)
	log.Infof("Found %d database(s) configured", len(cfg.EnabledDatabases()))

	// Initialize OAuth service if Google Drive is enabled
	var oauthService OAuthService
	if cfg.HasUploadTarget("gdrive") {
		oauthService, err = NewGoogleOAuthService(log, "client_secret.json")
		if err != nil {
			log.Errorf("Failed to initialize Google Drive OAuth service: %v", err)
		} else {
			log.Infof("Google Drive OAuth service initialized")
			addr := fmt.Sprintf(":%d", cfg.App.Port)
			if err := oauthService.StartAuthServer(ctx, addr); err != nil {
				log.Errorf("Failed to start OAuth server: %v", err)
			}
		}
	}

	comp := compressor.NewGzip()
	uploadTargets := initializeUploadTargets(cfg, log, oauthService)
	backupJobs := initializeBackupJobs(cfg, uploadTargets, comp, log)

	if len(backupJobs) == 0 {
		return nil, fmt.Errorf("no enabled databases found")
	}

	cleanupUC := usecase.NewCleanup(uploadTargets, log, cfg.Backup.RetentionDays)
	sched := scheduler.New()

	return &App{
		config:        cfg,
		logger:        log,
		scheduler:     sched,
		uploadTargets: uploadTargets,
		backupJobs:    backupJobs,
		cleanupUC:     cleanupUC,
		oauthService:  oauthService,
	}, nil
}

// Run starts the application and its scheduled jobs.
func (a *App) Run(ctx context.Context) error {
	a.logger.Infof("Application started with %d backup job(s)", len(a.backupJobs))

	for _, job := range a.backupJobs {
		dbName := job.DatabaseName
		backupUC := job.BackupUC

		if err := a.scheduler.AddJob(job.Schedule, func(ctx context.Context) error {
			a.logger.Infof("=== Triggered scheduled backup for %s ===", dbName)
			return backupUC.Execute(ctx)
		}); err != nil {
			return fmt.Errorf("failed to schedule backup for %s: %w", dbName, err)
		}
	}

	cleanupSchedule := "0 0 3 * * *"
	a.logger.Infof("Scheduling cleanup: %s", cleanupSchedule)

	if err := a.scheduler.AddJob(cleanupSchedule, a.cleanupUC.Execute); err != nil {
		return fmt.Errorf("failed to schedule cleanup: %w", err)
	}

	a.scheduler.Start()
	a.logger.Infof("Scheduler started successfully")
	a.logger.Infof("Backup destinations: %d remote target(s)", len(a.uploadTargets))

	<-ctx.Done()
	return nil
}

// Shutdown gracefully stops the application.
func (a *App) Shutdown(ctx context.Context) {
	a.logger.Infof("Shutting down application...")
	a.scheduler.Stop()

	if a.oauthService != nil {
		if err := a.oauthService.Shutdown(ctx); err != nil {
			a.logger.Errorf("Failed to shutdown OAuth service: %v", err)
		}
	}

	a.logger.Close()
}

// initializeUploadTargets creates upload targets based on configuration.
func initializeUploadTargets(cfg *config.Config, log *logger.Logger, oauthService OAuthService) []usecase.UploadTarget {
	var targets []usecase.UploadTarget

	for _, targetCfg := range cfg.EnabledUploadTargets() {
		var stor domain.Storage
		var err error

		switch targetCfg.Type {
		case "gdrive":
			if oauthService == nil {
				log.Errorf("Google Drive OAuth service not initialized for target: %s", targetCfg.Type)
				continue
			}
			stor, err = storage.NewGDrive(context.Background(), &targetCfg, oauthService.GetConfig(), log)
			if err != nil {
				log.Errorf("Failed to initialize Google Drive: %v", err)
				continue
			}
			log.Infof("✓ Google Drive upload enabled")

		case "s3":
			stor, err = storage.NewS3(&targetCfg)
			if err != nil {
				log.Errorf("Failed to initialize S3: %v", err)
				continue
			}
			log.Infof("✓ AWS S3 upload enabled (bucket: %s)", targetCfg.Bucket)

		case "telegram":
			stor, err = storage.NewTelegram(&targetCfg)
			if err != nil {
				log.Errorf("Failed to initialize Telegram: %v", err)
				continue
			}
			log.Infof("✓ Telegram upload enabled")

		case "local":
			stor, err = storage.NewLocal(targetCfg.Path)
			if err != nil {
				log.Errorf("Failed to initialize Local: %v", err)
				continue
			}
			log.Infof("✓ Local upload enabled")

		default:
			log.Warnf("Unknown upload target type: %s", targetCfg.Type)
			continue
		}

		targets = append(targets, usecase.UploadTarget{
			Name:    targetCfg.Type,
			Storage: stor,
		})
	}

	return targets
}

// initializeBackupJobs creates backup jobs based on configuration.
func initializeBackupJobs(
	cfg *config.Config,
	uploadTargets []usecase.UploadTarget,
	comp domain.Compressor,
	log *logger.Logger,
) []domain.BackupJob {
	var jobs []domain.BackupJob

	for _, dbCfg := range cfg.EnabledDatabases() {
		var db domain.Database

		switch dbCfg.Type {
		case "mysql":
			db = database.NewMySQL(&dbCfg)
		default:
			log.Warnf("Unsupported database type: %s for %s", dbCfg.Type, dbCfg.Name)
			continue
		}

		ctx := context.Background()
		if err := db.Ping(ctx); err != nil {
			log.Errorf("Failed to connect to %s: %v", dbCfg.Name, err)
			continue
		}
		log.Infof("✓ Connected to %s (%s)", dbCfg.Name, dbCfg.Type)

		backupUC := usecase.NewBackup(
			db,
			uploadTargets,
			comp,
			log,
			cfg.Backup.Compress,
		)

		jobs = append(jobs, domain.BackupJob{
			DatabaseName: dbCfg.Name,
			Schedule:     dbCfg.Schedule,
			Database:     db,
			BackupUC:     backupUC,
		})

		log.Infof("✓ Scheduled backup for %s: %s", dbCfg.Name, dbCfg.Schedule)
	}

	return jobs
}
