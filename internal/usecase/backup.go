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

type Backup struct {
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
) *Backup {
	return &Backup{
		db:            db,
		localStorage:  localStorage,
		uploadTargets: uploadTargets,
		compressor:    compressor,
		logger:        logger,
		compress:      compress,
	}
}

func (uc *Backup) Execute(ctx context.Context) error {
	start := time.Now()
	dbName := uc.db.GetName()
	uc.logger.Infof("[%s] Starting backup...", dbName)

	if err := uc.db.Ping(ctx); err != nil {
		return fmt.Errorf("database ping: %w", err)
	}

	filename := uc.generateFilename()
	tempPath := filepath.Join(os.TempDir(), filename)

	uc.logger.Infof("[%s] Creating backup to: %s", dbName, tempPath)
	if err := uc.db.Backup(ctx, tempPath); err != nil {
		return fmt.Errorf("backup: %w", err)
	}
	defer os.Remove(tempPath)

	fileInfo, err := os.Stat(tempPath)
	if err != nil {
		return fmt.Errorf("stat backup file: %w", err)
	}

	uc.logger.Infof("[%s] Backup created, size: %.2f MB",
		dbName, float64(fileInfo.Size())/(1024*1024))

	finalPath, finalFilename := tempPath, filename

	if uc.compress {
		finalPath, finalFilename, err = uc.compressBackup(tempPath, filename, fileInfo.Size())
		if err != nil {
			return err
		}
		defer os.Remove(finalPath)
	}

	if err := uc.uploadBackup(ctx, finalPath, finalFilename); err != nil {
		return err
	}

	uc.logger.Infof("[%s] Backup completed in %s: %s",
		dbName, time.Since(start).Round(time.Second), finalFilename)

	return nil
}

func (uc *Backup) generateFilename() string {
	timestamp := time.Now().Format("20060102_150405")
	baseFilename := fmt.Sprintf("%s_%s_%s", uc.db.GetName(), uc.db.GetType(), timestamp)

	ext := map[string]string{
		"mysql":      ".sql",
		"postgresql": ".dump",
		"mongodb":    ".archive",
	}[uc.db.GetType()]

	if ext == "" {
		ext = ".backup"
	}

	return baseFilename + ext
}

func (uc *Backup) compressBackup(tempPath, filename string, originalSize int64) (string, string, error) {
	dbName := uc.db.GetName()
	compressedFilename := filename + ".gz"
	compressedPath := filepath.Join(os.TempDir(), compressedFilename)

	uc.logger.Infof("[%s] Compressing backup...", dbName)
	if err := uc.compressor.Compress(tempPath, compressedPath); err != nil {
		return "", "", fmt.Errorf("compression: %w", err)
	}

	compressedInfo, _ := os.Stat(compressedPath)
	uc.logger.Infof("[%s] Compression complete, size: %.2f MB (%.1f%% of original)",
		dbName,
		float64(compressedInfo.Size())/(1024*1024),
		float64(compressedInfo.Size())/float64(originalSize)*100)

	return compressedPath, compressedFilename, nil
}

func (uc *Backup) uploadBackup(ctx context.Context, filePath, filename string) error {
	dbName := uc.db.GetName()

	uc.logger.Infof("[%s] Uploading to local storage...", dbName)
	if err := uc.localStorage.Upload(ctx, filePath, filename); err != nil {
		return fmt.Errorf("local upload: %w", err)
	}
	uc.logger.Infof("[%s] Successfully uploaded to local storage", dbName)

	if len(uc.uploadTargets) > 0 {
		uc.uploadToTargets(ctx, filePath, filename)
	}

	return nil
}

func (uc *Backup) uploadToTargets(ctx context.Context, filePath, filename string) {
	var wg sync.WaitGroup
	dbName := uc.db.GetName()

	for _, target := range uc.uploadTargets {
		wg.Add(1)
		go func(t UploadTarget) {
			defer wg.Done()

			uc.logger.Infof("[%s] Uploading to %s...", dbName, t.Name)
			if err := t.Storage.Upload(ctx, filePath, filename); err != nil {
				uc.logger.Errorf("[%s] Failed to upload to %s: %v", dbName, t.Name, err)
			} else {
				uc.logger.Infof("[%s] Successfully uploaded to %s", dbName, t.Name)
			}
		}(target)
	}

	wg.Wait()
}
