package database

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/semmidev/phylax/internal/config"
)

type MongoDBDatabase struct {
	config *config.DatabaseConfig
}

func NewMongoDB(cfg *config.DatabaseConfig) *MongoDBDatabase {
	return &MongoDBDatabase{config: cfg}
}

func (m *MongoDBDatabase) Backup(ctx context.Context, outputPath string) error {
	uri := fmt.Sprintf("mongodb://%s:%s@%s:%d/%s",
		m.config.Username,
		m.config.Password,
		m.config.Host,
		m.config.Port,
		m.config.Database,
	)

	if m.config.AuthDatabase != "" {
		uri += fmt.Sprintf("?authSource=%s", m.config.AuthDatabase)
	}

	args := []string{
		fmt.Sprintf("--uri=%s", uri),
		fmt.Sprintf("--archive=%s", outputPath),
		"--gzip",
	}

	cmd := exec.CommandContext(ctx, "mongodump", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mongodump failed: %w, output: %s", err, string(output))
	}

	return nil
}

func (m *MongoDBDatabase) GetName() string {
	return m.config.Name
}

func (m *MongoDBDatabase) GetType() string {
	return "mongodb"
}

func (m *MongoDBDatabase) Ping(ctx context.Context) error {
	uri := fmt.Sprintf("mongodb://%s:%s@%s:%d/%s",
		m.config.Username,
		m.config.Password,
		m.config.Host,
		m.config.Port,
		m.config.Database,
	)

	cmd := exec.CommandContext(ctx, "mongosh", uri, "--eval", "db.runCommand({ ping: 1 })")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mongodb ping failed: %w", err)
	}

	return nil
}
