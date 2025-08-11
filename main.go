package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"

	"github.com/ch00k/oar/internal/app"
	"github.com/ch00k/oar/logging"
	"github.com/ch00k/oar/services"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	// Initialize configuration for web app
	config, err := services.NewConfigForWebApp()
	if err != nil {
		log.Fatalf("Failed to initialize configuration: %v", err)
	}

	// Initialize logging with config
	logging.InitLogging(config.LogLevel)

	// Initialize application with config
	if err := app.InitializeWithConfig(config); err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)

	// Serve static files
	r.Handle("/assets/*", http.StripPrefix("/assets/", http.FileServer(http.Dir("./ui/assets/"))))

	// Initialize handlers
	// TODO

	// Register routes
	// TODO

	// Health check endpoint
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			slog.Error("Failed to write health check response",
				"layer", "main",
				"operation", "health_check",
				"error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	})

	address := fmt.Sprintf("%s:%d", config.HTTPHost, config.HTTPPort)
	slog.Info("Server starting", "address", fmt.Sprintf("http://%s", address))
	if err := http.ListenAndServe(address, r); err != nil {
		panic(err)
	}
}
