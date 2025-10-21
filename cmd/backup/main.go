// cmd/backup/main.go
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/semmidev/phylax/internal/app"
	"github.com/semmidev/phylax/internal/config"
	"github.com/semmidev/phylax/internal/infrastructure/logger"
)

// main is the entry point for the backup application.
func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// run initializes and starts the application, handling configuration and signals.
func run() error {
	// Parse command-line flags
	configPath := flag.String("config", "configs/config.yaml", "path to configuration file (YAML)")
	flag.Parse()

	// Allow config path override via environment variable
	if envConfig := os.Getenv("PHYLAX_CONFIG"); envConfig != "" {
		*configPath = envConfig
	}

	// Initialize context with signal handling for graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize logger early for error reporting
	log, err := logger.New(cfg.App.LogLevel, cfg.App.LogFile)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer log.Close()

	log.Infof("Starting application with config: %s", *configPath)

	// Initialize the application
	application, err := app.New(ctx, cfg)
	if err != nil {
		log.Errorf("Failed to initialize application: %v", err)
		return fmt.Errorf("failed to initialize application: %w", err)
	}
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		application.Shutdown(shutdownCtx)
		log.Infof("Application shutdown complete")
	}()

	// Run the application
	log.Infof("Running application...")
	if err := application.Run(ctx); err != nil {
		log.Errorf("Application run failed: %v", err)
		return fmt.Errorf("failed to run application: %w", err)
	}

	log.Infof("Application stopped gracefully")
	return nil
}
