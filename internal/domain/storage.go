package domain

import (
	"context"
	"time"
)

type Storage interface {
	Upload(ctx context.Context, localPath string, remoteName string) error
	List(ctx context.Context) ([]string, error)
	Delete(ctx context.Context, remoteName string) error
	GetOldFiles(ctx context.Context, cutoffTime time.Time) ([]string, error)
}
