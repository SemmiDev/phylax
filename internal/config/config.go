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
	LogLevel string `mapstructure:"log_level"`
	LogFile  string `mapstructure:"log_file"`
}

type DatabaseConfig struct {
	Name     string `mapstructure:"name"`
	Type     string `mapstructure:"type"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`
	Enabled  bool   `mapstructure:"enabled"`
	Schedule string `mapstructure:"schedule"`

	// PostgreSQL specific
	SSLMode string `mapstructure:"ssl_mode"`

	// MongoDB specific
	AuthDatabase string `mapstructure:"auth_database"`
}

type BackupConfig struct {
	LocalPath     string         `mapstructure:"local_path"`
	RetentionDays int            `mapstructure:"retention_days"`
	Compress      bool           `mapstructure:"compress"`
	UploadTargets []UploadTarget `mapstructure:"upload_targets"`
}

type UploadTarget struct {
	Type    string `mapstructure:"type"`
	Enabled bool   `mapstructure:"enabled"`

	// Google Drive
	CredentialsFile string `mapstructure:"credentials_file"`
	FolderID        string `mapstructure:"folder_id"`

	// AWS S3
	Region    string `mapstructure:"region"`
	Bucket    string `mapstructure:"bucket"`
	AccessKey string `mapstructure:"access_key"`
	SecretKey string `mapstructure:"secret_key"`
	Prefix    string `mapstructure:"prefix"`

	// Telegram
	BotToken   string `mapstructure:"bot_token"`
	ChatID     string `mapstructure:"chat_id"`
	SendFile   bool   `mapstructure:"send_file"`
	NotifyOnly bool   `mapstructure:"notify_only"`
}

func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	v.SetDefault("app.name", "phylax")
	v.SetDefault("app.log_level", "info")
	v.SetDefault("backup.retention_days", 7)
	v.SetDefault("backup.compress", true)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
	if len(c.Databases) == 0 {
		return fmt.Errorf("at least one database configuration is required")
	}

	for i, db := range c.Databases {
		if db.Name == "" {
			return fmt.Errorf("database[%d]: name is required", i)
		}
		if db.Type == "" {
			return fmt.Errorf("database[%d]: type is required", i)
		}
		if db.Host == "" {
			return fmt.Errorf("database[%d]: host is required", i)
		}
		if db.Enabled && db.Schedule == "" {
			return fmt.Errorf("database[%d]: schedule is required when enabled", i)
		}
	}

	if c.Backup.LocalPath == "" {
		return fmt.Errorf("backup.local_path is required")
	}

	return nil
}

func (c *Config) GetEnabledDatabases() []DatabaseConfig {
	var enabled []DatabaseConfig
	for _, db := range c.Databases {
		if db.Enabled {
			enabled = append(enabled, db)
		}
	}
	return enabled
}

func (c *Config) GetEnabledUploadTargets() []UploadTarget {
	var enabled []UploadTarget
	for _, target := range c.Backup.UploadTargets {
		if target.Enabled {
			enabled = append(enabled, target)
		}
	}
	return enabled
}
