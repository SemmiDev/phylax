package domain

import "context"

type Database interface {
	Backup(ctx context.Context, outputPath string) error
	GetName() string
	GetType() string
	Ping(ctx context.Context) error
}
