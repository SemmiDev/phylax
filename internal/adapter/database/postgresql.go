package database

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/semmidev/phylax/internal/config"
)

type PostgreSQLDatabase struct {
	config *config.DatabaseConfig
}

func NewPostgreSQL(cfg *config.DatabaseConfig) *PostgreSQLDatabase {
	return &PostgreSQLDatabase{config: cfg}
}

func (p *PostgreSQLDatabase) Backup(ctx context.Context, outputPath string) error {
	// Set PGPASSWORD environment variable
	cmd := exec.CommandContext(ctx, "pg_dump",
		fmt.Sprintf("--host=%s", p.config.Host),
		fmt.Sprintf("--port=%d", p.config.Port),
		fmt.Sprintf("--username=%s", p.config.Username),
		"--format=custom",
		"--compress=9",
		"--verbose",
		fmt.Sprintf("--file=%s", outputPath),
		p.config.Database,
	)

	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", p.config.Password))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pg_dump failed: %w, output: %s", err, string(output))
	}

	return nil
}

func (p *PostgreSQLDatabase) GetName() string {
	return p.config.Name
}

func (p *PostgreSQLDatabase) GetType() string {
	return "postgresql"
}

func (p *PostgreSQLDatabase) Ping(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "psql",
		fmt.Sprintf("--host=%s", p.config.Host),
		fmt.Sprintf("--port=%d", p.config.Port),
		fmt.Sprintf("--username=%s", p.config.Username),
		"--dbname=postgres",
		"-c", "SELECT 1",
	)

	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", p.config.Password))

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("postgresql ping failed: %w", err)
	}

	return nil
}
