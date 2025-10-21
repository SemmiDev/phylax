package usecase

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/semmidev/phylax/internal/domain"
)

type BackupUseCase struct {
	db            domain.Database
	localStorage  LocalStorage
	uploadTargets []UploadTarget
	compressor    domain.Compressor
	logger        Logger
	compress      bool
}

type UploadTarget struct {
	Name    string
	Storage domain.Storage
}

type LocalStorage interface {
	domain.Storage
	GetPath(filename string) string
}

type Logger interface {
	Infof(template string, args ...interface{})
	Errorf(template string, args ...interface{})
	Warnf(template string, args ...interface{})
}

func NewBackup(
	db domain.Database,
	localStorage LocalStorage,
	uploadTargets []UploadTarget,
	compressor domain.Compressor,
	logger Logger,
	compress bool,
) *BackupUseCase {
	return &BackupUseCase{
		db:            db,
		localStorage:  localStorage,
		uploadTargets: uploadTargets,
		compressor:    compressor,
		logger:        logger,
		compress:      compress,
	}
}

func (uc *BackupUseCase) Execute(ctx context.Context) error {
	startTime := time.Now()
	uc.logger.Infof("[%s] Starting backup...", uc.db.GetName())

	// Check database connectivity
	if err := uc.db.Ping(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	// Generate backup filename
	timestamp := time.Now().Format("20060102_150405")
	baseFilename := fmt.Sprintf("%s_%s_%s", uc.db.GetName(), uc.db.GetType(), timestamp)

	var extension string
	switch uc.db.GetType() {
	case "mysql":
		extension = ".sql"
	case "postgresql":
		extension = ".dump"
	case "mongodb":
		extension = ".archive"
	default:
		extension = ".backup"
	}

	filename := baseFilename + extension
	tempPath := filepath.Join(os.TempDir(), filename)

	// Perform database backup
	uc.logger.Infof("[%s] Creating backup to: %s", uc.db.GetName(), tempPath)
	if err := uc.db.Backup(ctx, tempPath); err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}
	defer os.Remove(tempPath)

	// Get file size
	fileInfo, err := os.Stat(tempPath)
	if err != nil {
		return fmt.Errorf("failed to stat backup file: %w", err)
	}

	uc.logger.Infof("[%s] Backup created successfully, size: %.2f MB",
		uc.db.GetName(),
		float64(fileInfo.Size())/(1024*1024))

	finalPath := tempPath
	finalFilename := filename

	// Compress if enabled
	if uc.compress {
		compressedFilename := filename + ".gz"
		compressedPath := filepath.Join(os.TempDir(), compressedFilename)

		uc.logger.Infof("[%s] Compressing backup...", uc.db.GetName())
		if err := uc.compressor.Compress(tempPath, compressedPath); err != nil {
			return fmt.Errorf("compression failed: %w", err)
		}
		defer os.Remove(compressedPath)

		compressedInfo, _ := os.Stat(compressedPath)
		uc.logger.Infof("[%s] Compression complete, size: %.2f MB (%.1f%% of original)",
			uc.db.GetName(),
			float64(compressedInfo.Size())/(1024*1024),
			float64(compressedInfo.Size())/float64(fileInfo.Size())*100)

		finalPath = compressedPath
		finalFilename = compressedFilename
	}

	// Upload to local storage
	uc.logger.Infof("[%s] Uploading to local storage...", uc.db.GetName())
	if err := uc.localStorage.Upload(ctx, finalPath, finalFilename); err != nil {
		return fmt.Errorf("local upload failed: %w", err)
	}
	uc.logger.Infof("[%s] Successfully uploaded to local storage", uc.db.GetName())

	// Upload to all enabled targets in parallel
	if len(uc.uploadTargets) > 0 {
		uc.uploadToTargets(ctx, finalPath, finalFilename)
	}

	duration := time.Since(startTime)
	uc.logger.Infof("[%s] Backup completed successfully in %s: %s",
		uc.db.GetName(),
		duration.Round(time.Second),
		finalFilename)

	return nil
}

func (uc *BackupUseCase) uploadToTargets(ctx context.Context, filePath, filename string) {
	var wg sync.WaitGroup

	for _, target := range uc.uploadTargets {
		wg.Add(1)
		go func(t UploadTarget) {
			defer wg.Done()

			uc.logger.Infof("[%s] Uploading to %s...", uc.db.GetName(), t.Name)
			if err := t.Storage.Upload(ctx, filePath, filename); err != nil {
				uc.logger.Errorf("[%s] Failed to upload to %s: %v", uc.db.GetName(), t.Name, err)
			} else {
				uc.logger.Infof("[%s] Successfully uploaded to %s", uc.db.GetName(), t.Name)
			}
		}(target)
	}

	wg.Wait()
}
