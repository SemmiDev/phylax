package domain

import (
	"context"
	"time"
)

type Backup struct {
	ID           string
	Filename     string
	FilePath     string
	Size         int64
	Compressed   bool
	CreatedAt    time.Time
	DatabaseName string
}

type BackupMetadata struct {
	Database  string
	Timestamp time.Time
	Size      int64
}

type BackupJob struct {
	DatabaseName string
	Schedule     string
	Database     Database
	BackupUC     BackupExecutor
}

type BackupExecutor interface {
	Execute(ctx context.Context) error
}
