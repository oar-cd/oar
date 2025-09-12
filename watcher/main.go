package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/oar-cd/oar/internal/app"
	"github.com/oar-cd/oar/logging"
	"github.com/oar-cd/oar/services"
	"github.com/oar-cd/oar/watcher/service"
)

func main() {
	// Initialize configuration for watcher app
	config, err := services.NewConfigForWebApp()
	if err != nil {
		log.Fatalf("Failed to initialize configuration: %v", err)
	}

	// Initialize logging with config
	logging.InitLogging(config.LogLevel)

	slog.Info("Starting Oar Watcher Service")

	// Initialize application with config
	if err := app.InitializeWithConfig(config); err != nil {
		slog.Error("Failed to initialize application", "error", err)
		os.Exit(1)
	}

	// Initialize watcher service
	watcherService := service.NewWatcherService(
		app.GetProjectService(),
		app.GetGitService(),
		config.PollInterval, // Use configured poll interval
	)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals in background
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		slog.Info("Shutdown signal received, stopping watcher service")
		cancel()
	}()

	// Run watcher service in main thread
	if err := watcherService.Start(ctx); err != nil {
		slog.Error("Watcher service failed", "error", err)
		os.Exit(1)
	}

	slog.Info("Watcher service stopped")
}
