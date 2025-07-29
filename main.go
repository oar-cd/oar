package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"

	"github.com/ch00k/oar/internal/app"
	"github.com/ch00k/oar/internal/handlers"
	"github.com/ch00k/oar/logging"
	"github.com/ch00k/oar/services"
	"github.com/ch00k/oar/ui/pages"
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
	projectHandlers := handlers.NewProjectHandlers(app.GetProjectService())
	discoveryHandlers := handlers.NewDiscoveryHandlers(app.GetDiscoveryService())
	bootstrapHandlers := handlers.NewBootstrapHandlers(app.GetProjectService())

	// Register routes
	projectHandlers.RegisterRoutes(r)
	discoveryHandlers.RegisterRoutes(r)
	bootstrapHandlers.RegisterRoutes(r)

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

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		projectService := app.GetProjectService()
		projects, err := projectService.List()
		if err != nil {
			slog.Error("Application operation failed",
				"layer", "main",
				"operation", "list_projects",
				"error", err)
			projects = []*services.Project{} // Empty slice on error
		}

		component := pages.Home(projects)
		if err := component.Render(r.Context(), w); err != nil {
			http.Error(w, "Failed to render home page", http.StatusInternalServerError)
		}
	})

	address := fmt.Sprintf("%s:%d", config.HTTPHost, config.HTTPPort)
	slog.Info("Server starting", "address", fmt.Sprintf("http://%s", address))
	if err := http.ListenAndServe(address, r); err != nil {
		panic(err)
	}
}
