package domain

import "context"

type Database interface {
	Backup(ctx context.Context, outputPath string) error
	Name() string
	Type() string
	Ping(ctx context.Context) error
}
