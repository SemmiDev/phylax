package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/semmidev/phylax/internal/app"
	"github.com/semmidev/phylax/internal/config"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	application, err := app.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize app: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := application.Run(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "Application error: %v\n", err)
			cancel()
		}
	}()

	sig := <-sigChan
	fmt.Printf("\nReceived signal %v, shutting down gracefully...\n", sig)
	cancel()

	application.Shutdown()
	fmt.Println("Shutdown complete")
}
