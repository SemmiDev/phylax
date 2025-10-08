package database

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/semmidev/phylax/internal/config"
)

type MySQLDatabase struct {
	config *config.DatabaseConfig
}

func NewMySQL(cfg *config.DatabaseConfig) *MySQLDatabase {
	return &MySQLDatabase{config: cfg}
}

func (m *MySQLDatabase) Backup(ctx context.Context, outputPath string) error {
	args := []string{
		fmt.Sprintf("--host=%s", m.config.Host),
		fmt.Sprintf("--port=%d", m.config.Port),
		fmt.Sprintf("--user=%s", m.config.Username),
		fmt.Sprintf("--password=%s", m.config.Password),
		"--single-transaction",
		"--quick",
		"--lock-tables=false",
		"--routines",
		"--triggers",
		"--events",
		fmt.Sprintf("--result-file=%s", outputPath),
		m.config.Database,
	}

	cmd := exec.CommandContext(ctx, "mysqldump", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mysqldump failed: %w, output: %s", err, string(output))
	}

	return nil
}

func (m *MySQLDatabase) GetName() string {
	return m.config.Name
}

func (m *MySQLDatabase) GetType() string {
	return "mysql"
}

func (m *MySQLDatabase) Ping(ctx context.Context) error {
	args := []string{
		fmt.Sprintf("--host=%s", m.config.Host),
		fmt.Sprintf("--port=%d", m.config.Port),
		fmt.Sprintf("--user=%s", m.config.Username),
		fmt.Sprintf("--password=%s", m.config.Password),
		"-e", "SELECT 1",
	}

	cmd := exec.CommandContext(ctx, "mysql", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mysql ping failed: %w", err)
	}

	return nil
}
