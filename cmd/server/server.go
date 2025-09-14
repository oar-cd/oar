// Package server implements the unified server command for running both web and watcher services.
package server

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/oar-cd/oar/app"
	"github.com/oar-cd/oar/logging"
	"github.com/oar-cd/oar/services"
	"github.com/oar-cd/oar/watcher"
	"github.com/oar-cd/oar/web/routes"
	"github.com/spf13/cobra"
)

// NewCmdServer creates a command to run both web and watcher services
func NewCmdServer() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Run Oar server (web interface + deployment watcher)",
		Long:  "Starts both the web interface and deployment watcher in a single process",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, _ := cmd.Flags().GetString("config")
			return runServer(configPath)
		},
	}

	cmd.Flags().StringP("config", "c", "", "Path to configuration file")
	return cmd
}

// runServer runs both web and watcher services
func runServer(configPath string) error {
	// Initialize configuration from YAML file
	config, err := services.NewConfigFromFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to initialize configuration: %w", err)
	}

	// Initialize logging
	logging.InitLogging(config.LogLevel)

	slog.Info("Starting Oar Server (web + watcher)")

	// Initialize application
	if err := app.InitializeWithConfig(config); err != nil {
		return fmt.Errorf("failed to initialize application: %w", err)
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	go handleShutdown(cancel)

	// Start watcher service in background
	go func() {
		if err := startWatcherService(ctx); err != nil {
			slog.Error("Watcher service failed", "error", err)
			cancel() // Trigger shutdown
		}
	}()

	// Start web server (blocks until shutdown)
	return startWebServer(ctx, config)
}

// startWebServer starts the HTTP server
func startWebServer(ctx context.Context, config *services.Config) error {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	// Serve static files
	r.Handle("/assets/*", http.StripPrefix("/assets/", http.FileServer(http.Dir("./web/assets/"))))

	// Register all routes
	routes.RegisterHomeRoutes(r)
	routes.RegisterProjectRoutes(r)
	routes.RegisterUtilityRoutes(r)

	// Create HTTP server
	address := fmt.Sprintf("%s:%d", config.HTTPHost, config.HTTPPort)
	server := &http.Server{
		Addr:    address,
		Handler: r,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Web server starting on http://%s", address)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Web server failed", "error", err)
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()

	// Graceful shutdown
	slog.Info("Shutting down web server")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("web server shutdown failed: %w", err)
	}

	slog.Info("Web server stopped")
	return nil
}

// startWatcherService starts the watcher service
func startWatcherService(ctx context.Context) error {
	// Get configuration (should already be initialized)
	config, err := services.NewConfigForWebApp()
	if err != nil {
		return fmt.Errorf("failed to get configuration: %w", err)
	}

	// Initialize watcher service
	watcherService := watcher.NewWatcherService(
		app.GetProjectService(),
		app.GetGitService(),
		config.PollInterval,
	)

	// Run watcher service
	if err := watcherService.Start(ctx); err != nil {
		return fmt.Errorf("watcher service failed: %w", err)
	}

	slog.Info("Watcher service stopped")
	return nil
}

// handleShutdown handles OS signals for graceful shutdown
func handleShutdown(cancel context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	slog.Info("Shutdown signal received")
	cancel()
}
