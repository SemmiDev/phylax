package usecase

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"
)

type CleanupUseCase struct {
	localStorage  localStorage
	uploadTargets []UploadTarget
	logger        Logger
	retentionDays int
}

func NewCleanup(
	localStorage localStorage,
	uploadTargets []UploadTarget,
	logger Logger,
	retentionDays int,
) *CleanupUseCase {
	return &CleanupUseCase{
		localStorage:  localStorage,
		uploadTargets: uploadTargets,
		logger:        logger,
		retentionDays: retentionDays,
	}
}

func (uc *CleanupUseCase) Execute(ctx context.Context) error {
	uc.logger.Infof("Starting cleanup, retention: %d days", uc.retentionDays)

	cutoffTime := time.Now().AddDate(0, 0, -uc.retentionDays)

	// Cleanup local storage
	if err := uc.cleanupLocal(ctx, cutoffTime); err != nil {
		uc.logger.Errorf("Local cleanup failed: %v", err)
	}

	// Cleanup all upload targets in parallel
	if len(uc.uploadTargets) > 0 {
		uc.cleanupTargets(ctx, cutoffTime)
	}

	uc.logger.Infof("Cleanup completed")
	return nil
}

func (uc *CleanupUseCase) cleanupLocal(ctx context.Context, cutoffTime time.Time) error {
	files, err := uc.localStorage.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list files: %w", err)
	}

	deletedCount := 0
	for _, filename := range files {
		filePath := uc.localStorage.GetPath(filename)
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			uc.logger.Warnf("Failed to stat file %s: %v", filename, err)
			continue
		}

		if fileInfo.ModTime().Before(cutoffTime) {
			uc.logger.Infof("Deleting old backup from local: %s (age: %s)",
				filename,
				time.Since(fileInfo.ModTime()).Round(24*time.Hour))

			if err := uc.localStorage.Delete(ctx, filename); err != nil {
				uc.logger.Errorf("Failed to delete %s: %v", filename, err)
			} else {
				deletedCount++
			}
		}
	}

	uc.logger.Infof("Deleted %d old backup(s) from local storage", deletedCount)
	return nil
}

func (uc *CleanupUseCase) cleanupTargets(ctx context.Context, cutoffTime time.Time) {
	var wg sync.WaitGroup

	for _, target := range uc.uploadTargets {
		wg.Add(1)
		go func(t UploadTarget) {
			defer wg.Done()

			if err := uc.cleanupTarget(ctx, t, cutoffTime); err != nil {
				uc.logger.Errorf("Cleanup failed for %s: %v", t.Name, err)
			}
		}(target)
	}

	wg.Wait()
}

func (uc *CleanupUseCase) cleanupTarget(ctx context.Context, target UploadTarget, cutoffTime time.Time) error {
	// Try to get old files with timestamp support
	type OldFileGetter interface {
		GetOldFiles(ctx context.Context, cutoffTime time.Time) ([]string, error)
	}

	deletedCount := 0

	if getter, ok := target.Storage.(OldFileGetter); ok {
		files, err := getter.GetOldFiles(ctx, cutoffTime)
		if err != nil {
			return fmt.Errorf("failed to list old files: %w", err)
		}

		for _, filename := range files {
			uc.logger.Infof("Deleting old backup from %s: %s", target.Name, filename)

			if err := target.Storage.Delete(ctx, filename); err != nil {
				uc.logger.Errorf("Failed to delete %s from %s: %v", filename, target.Name, err)
			} else {
				deletedCount++
			}
		}
	} else {
		// Fallback: list all and parse timestamp
		files, err := target.Storage.List(ctx)
		if err != nil {
			return fmt.Errorf("failed to list files: %w", err)
		}

		for _, filename := range files {
			timestamp, err := extractTimestamp(filename)
			if err != nil {
				uc.logger.Warnf("Could not parse timestamp from %s: %v", filename, err)
				continue
			}

			if timestamp.Before(cutoffTime) {
				uc.logger.Infof("Deleting old backup from %s: %s", target.Name, filename)

				if err := target.Storage.Delete(ctx, filename); err != nil {
					uc.logger.Errorf("Failed to delete %s from %s: %v", filename, target.Name, err)
				} else {
					deletedCount++
				}
			}
		}
	}

	uc.logger.Infof("Deleted %d old backup(s) from %s", deletedCount, target.Name)
	return nil
}
