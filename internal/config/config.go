package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	App       AppConfig        `mapstructure:"app"`
	Databases []DatabaseConfig `mapstructure:"databases"`
	Backup    BackupConfig     `mapstructure:"backup"`
}

type AppConfig struct {
	Name     string `mapstructure:"name"`
	Port     int    `mapstructure:"port"`
	LogLevel string `mapstructure:"log_level"`
	LogFile  string `mapstructure:"log_file"`
}

type DatabaseConfig struct {
	Name         string `mapstructure:"name"`
	Type         string `mapstructure:"type"`
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	Username     string `mapstructure:"username"`
	Password     string `mapstructure:"password"`
	Database     string `mapstructure:"database"`
	Enabled      bool   `mapstructure:"enabled"`
	Schedule     string `mapstructure:"schedule"`
	SSLMode      string `mapstructure:"ssl_mode"`
	AuthDatabase string `mapstructure:"auth_database"`
}

type BackupConfig struct {
	RetentionDays int            `mapstructure:"retention_days"`
	Compress      bool           `mapstructure:"compress"`
	UploadTargets []UploadTarget `mapstructure:"upload_targets"`
}

type UploadTarget struct {
	Type            string `mapstructure:"type"`
	Path            string `mapstructure:"path"`
	RefreshToken    string `mapstructure:"refresh_token"`
	Enabled         bool   `mapstructure:"enabled"`
	CredentialsFile string `mapstructure:"credentials_file"`
	FolderID        string `mapstructure:"folder_id"`
	Region          string `mapstructure:"region"`
	Bucket          string `mapstructure:"bucket"`
	AccessKey       string `mapstructure:"access_key"`
	SecretKey       string `mapstructure:"secret_key"`
	Prefix          string `mapstructure:"prefix"`
	BotToken        string `mapstructure:"bot_token"`
	ChatID          string `mapstructure:"chat_id"`
	SendFile        bool   `mapstructure:"send_file"`
	NotifyOnly      bool   `mapstructure:"notify_only"`
}

func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	v.SetDefault("app.name", "phylax")
	v.SetDefault("app.log_level", "info")
	v.SetDefault("backup.retention_days", 14)
	v.SetDefault("backup.compress", true)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) validate() error {
	if len(c.Databases) == 0 {
		return fmt.Errorf("at least one database required")
	}

	for i, db := range c.Databases {
		if db.Name == "" {
			return fmt.Errorf("database[%d]: name required", i)
		}
		if db.Type == "" {
			return fmt.Errorf("database[%d]: type required", i)
		}
		if db.Host == "" {
			return fmt.Errorf("database[%d]: host required", i)
		}
		if db.Enabled && db.Schedule == "" {
			return fmt.Errorf("database[%d]: schedule required when enabled", i)
		}
	}

	return nil
}

func (c *Config) HasUploadTarget(targetType string) bool {
	for _, target := range c.EnabledUploadTargets() {
		if target.Type == targetType {
			return true
		}
	}
	return false
}

func (c *Config) EnabledDatabases() []DatabaseConfig {
	var enabled []DatabaseConfig
	for _, db := range c.Databases {
		if db.Enabled {
			enabled = append(enabled, db)
		}
	}
	return enabled
}

func (c *Config) EnabledUploadTargets() []UploadTarget {
	var enabled []UploadTarget
	for _, target := range c.Backup.UploadTargets {
		if target.Enabled {
			enabled = append(enabled, target)
		}
	}
	return enabled
}
