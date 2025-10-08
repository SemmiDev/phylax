package app

import (
	"context"
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

type App struct {
	config        *config.Config
	logger        *logger.Logger
	scheduler     *scheduler.Scheduler
	localStorage  *storage.LocalStorage
	uploadTargets []usecase.UploadTarget
	backupJobs    []domain.BackupJob
	cleanupUC     *usecase.CleanupUseCase
}

func New(cfg *config.Config) (*App, error) {
	// Initialize logger
	log, err := logger.New(cfg.App.LogLevel, cfg.App.LogFile)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	log.Infof("Starting %s", cfg.App.Name)
	log.Infof("Found %d database(s) configured", len(cfg.GetEnabledDatabases()))

	// Initialize local storage
	localStorage, err := storage.NewLocal(cfg.Backup.LocalPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize local storage: %w", err)
	}

	// Initialize compressor
	comp := compressor.NewGzip()

	// Initialize upload targets
	uploadTargets := initializeUploadTargets(cfg, log)

	// Initialize databases and backup jobs
	backupJobs := initializeBackupJobs(cfg, localStorage, uploadTargets, comp, log)

	if len(backupJobs) == 0 {
		return nil, fmt.Errorf("no enabled databases found")
	}

	// Initialize cleanup use case
	cleanupUC := usecase.NewCleanup(
		localStorage,
		uploadTargets,
		log,
		cfg.Backup.RetentionDays,
	)

	// Initialize scheduler
	sched := scheduler.New()

	return &App{
		config:        cfg,
		logger:        log,
		scheduler:     sched,
		localStorage:  localStorage,
		uploadTargets: uploadTargets,
		backupJobs:    backupJobs,
		cleanupUC:     cleanupUC,
	}, nil
}

func initializeUploadTargets(cfg *config.Config, log *logger.Logger) []usecase.UploadTarget {
	var targets []usecase.UploadTarget

	for _, targetCfg := range cfg.GetEnabledUploadTargets() {
		var stor domain.Storage
		var err error

		switch targetCfg.Type {
		case "gdrive":
			stor, err = storage.NewGDrive(&targetCfg)
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
			// Local storage is always enabled
			continue

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

func initializeBackupJobs(
	cfg *config.Config,
	localStorage *storage.LocalStorage,
	uploadTargets []usecase.UploadTarget,
	comp domain.Compressor,
	log *logger.Logger,
) []domain.BackupJob {
	var jobs []domain.BackupJob

	for _, dbCfg := range cfg.GetEnabledDatabases() {
		var db domain.Database

		switch dbCfg.Type {
		case "mysql":
			db = database.NewMySQL(&dbCfg)
		case "postgresql":
			db = database.NewPostgreSQL(&dbCfg)
		case "mongodb":
			db = database.NewMongoDB(&dbCfg)
		default:
			log.Warnf("Unsupported database type: %s for %s", dbCfg.Type, dbCfg.Name)
			continue
		}

		// Test connection
		ctx := context.Background()
		if err := db.Ping(ctx); err != nil {
			log.Errorf("Failed to connect to %s: %v", dbCfg.Name, err)
			continue
		}
		log.Infof("✓ Connected to %s (%s)", dbCfg.Name, dbCfg.Type)

		// Create backup use case for this database
		backupUC := usecase.NewBackup(
			db,
			localStorage,
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

func (a *App) Run(ctx context.Context) error {
	a.logger.Infof("Application started with %d backup job(s)", len(a.backupJobs))

	// Schedule all backup jobs
	for _, job := range a.backupJobs {
		backupUC := job.BackupUC
		dbName := job.DatabaseName

		if err := a.scheduler.AddJob(job.Schedule, func(ctx context.Context) error {
			a.logger.Infof("=== Triggered scheduled backup for %s ===", dbName)
			return backupUC.Execute(ctx)
		}); err != nil {
			return fmt.Errorf("failed to schedule backup for %s: %w", dbName, err)
		}
	}

	// Schedule cleanup job (daily at 3 AM)
	cleanupSchedule := "0 0 3 * * *"
	a.logger.Infof("Scheduling cleanup: %s", cleanupSchedule)

	if err := a.scheduler.AddJob(cleanupSchedule, a.cleanupUC.Execute); err != nil {
		return fmt.Errorf("failed to schedule cleanup: %w", err)
	}

	a.scheduler.Start()
	a.logger.Infof("Scheduler started successfully")
	a.logger.Infof("Backup destinations: local + %d remote target(s)", len(a.uploadTargets))

	// Keep running until context is cancelled
	<-ctx.Done()
	return nil
}

func (a *App) Shutdown() {
	a.logger.Infof("Shutting down application...")
	a.scheduler.Stop()
	a.logger.Close()
}
