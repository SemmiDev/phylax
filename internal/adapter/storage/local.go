package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type LocalStorage struct {
	basePath string
}

func NewLocal(basePath string) (*LocalStorage, error) {
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}
	return &LocalStorage{basePath: basePath}, nil
}

func (l *LocalStorage) Upload(ctx context.Context, localPath string, remoteName string) error {
	destPath := filepath.Join(l.basePath, remoteName)

	source, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open source: %w", err)
	}
	defer source.Close()

	dest, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create dest: %w", err)
	}
	defer dest.Close()

	if _, err := dest.ReadFrom(source); err != nil {
		return fmt.Errorf("failed to copy: %w", err)
	}

	return nil
}

func (l *LocalStorage) List(ctx context.Context) ([]string, error) {
	entries, err := os.ReadDir(l.basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}

	return files, nil
}

func (l *LocalStorage) Delete(ctx context.Context, remoteName string) error {
	filePath := filepath.Join(l.basePath, remoteName)
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

func (l *LocalStorage) GetOldFiles(ctx context.Context, cutoffTime time.Time) ([]string, error) {
	entries, err := os.ReadDir(l.basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var oldFiles []string
	for _, entry := range entries {
		if !entry.IsDir() {
			info, err := entry.Info()
			if err != nil {
				return nil, fmt.Errorf("failed to get file info for %s: %w", entry.Name(), err)
			}
			if info.ModTime().Before(cutoffTime) {
				oldFiles = append(oldFiles, entry.Name())
			}
		}
	}

	return oldFiles, nil
}

func (l *LocalStorage) GetPath(filename string) string {
	return filepath.Join(l.basePath, filename)
}
