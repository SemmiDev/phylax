package usecase

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"sync"
	"time"
)

type Cleanup struct {
	localStorage  LocalStorage
	uploadTargets []UploadTarget
	logger        Logger
	retentionDays int
}

func NewCleanup(
	localStorage LocalStorage,
	uploadTargets []UploadTarget,
	logger Logger,
	retentionDays int,
) *Cleanup {
	return &Cleanup{
		localStorage:  localStorage,
		uploadTargets: uploadTargets,
		logger:        logger,
		retentionDays: retentionDays,
	}
}

func (uc *Cleanup) Execute(ctx context.Context) error {
	uc.logger.Infof("Starting cleanup, retention: %d days", uc.retentionDays)

	cutoff := time.Now().AddDate(0, 0, -uc.retentionDays)

	if err := uc.cleanupLocal(ctx, cutoff); err != nil {
		uc.logger.Errorf("Local cleanup failed: %v", err)
	}

	if len(uc.uploadTargets) > 0 {
		uc.cleanupTargets(ctx, cutoff)
	}

	uc.logger.Infof("Cleanup completed")
	return nil
}

func (uc *Cleanup) cleanupLocal(ctx context.Context, cutoff time.Time) error {
	files, err := uc.localStorage.List(ctx)
	if err != nil {
		return fmt.Errorf("list files: %w", err)
	}

	deleted := 0
	for _, filename := range files {
		filePath := uc.localStorage.GetPath(filename)
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			uc.logger.Warnf("Failed to stat file %s: %v", filename, err)
			continue
		}

		if fileInfo.ModTime().Before(cutoff) {
			uc.logger.Infof("Deleting old backup from local: %s (age: %s)",
				filename, time.Since(fileInfo.ModTime()).Round(24*time.Hour))

			if err := uc.localStorage.Delete(ctx, filename); err != nil {
				uc.logger.Errorf("Failed to delete %s: %v", filename, err)
			} else {
				deleted++
			}
		}
	}

	uc.logger.Infof("Deleted %d old backup(s) from local storage", deleted)
	return nil
}

func (uc *Cleanup) cleanupTargets(ctx context.Context, cutoff time.Time) {
	var wg sync.WaitGroup

	for _, target := range uc.uploadTargets {
		wg.Add(1)
		go func(t UploadTarget) {
			defer wg.Done()

			if err := uc.cleanupTarget(ctx, t, cutoff); err != nil {
				uc.logger.Errorf("Cleanup failed for %s: %v", t.Name, err)
			}
		}(target)
	}

	wg.Wait()
}

func (uc *Cleanup) cleanupTarget(ctx context.Context, target UploadTarget, cutoff time.Time) error {
	files, err := target.Storage.GetOldFiles(ctx, cutoff)
	if err != nil {
		files, err = uc.fallbackListFiles(ctx, target, cutoff)
		if err != nil {
			return err
		}
	}

	deleted := 0
	for _, filename := range files {
		uc.logger.Infof("Deleting old backup from %s: %s", target.Name, filename)

		if err := target.Storage.Delete(ctx, filename); err != nil {
			uc.logger.Errorf("Failed to delete %s from %s: %v", filename, target.Name, err)
		} else {
			deleted++
		}
	}

	uc.logger.Infof("Deleted %d old backup(s) from %s", deleted, target.Name)
	return nil
}

func (uc *Cleanup) fallbackListFiles(ctx context.Context, target UploadTarget, cutoff time.Time) ([]string, error) {
	files, err := target.Storage.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list files: %w", err)
	}

	oldFiles := make([]string, 0)
	for _, filename := range files {
		timestamp, err := extractTimestamp(filename)
		if err != nil {
			uc.logger.Warnf("Could not parse timestamp from %s: %v", filename, err)
			continue
		}

		if timestamp.Before(cutoff) {
			oldFiles = append(oldFiles, filename)
		}
	}

	return oldFiles, nil
}

func extractTimestamp(filename string) (time.Time, error) {
	pattern := regexp.MustCompile(`(\d{8})_(\d{6})`)
	matches := pattern.FindStringSubmatch(filename)

	if len(matches) < 3 {
		return time.Time{}, fmt.Errorf("invalid filename format: no timestamp found")
	}

	timestampStr := matches[1] + "_" + matches[2]
	return time.Parse("20060102_150405", timestampStr)
}
