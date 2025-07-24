package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/ch00k/oar/internal/app"
	"github.com/ch00k/oar/internal/handlers"
	"github.com/ch00k/oar/services"
	"github.com/ch00k/oar/ui/pages"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	// Initialize application
	dataDir := os.Getenv("OAR_DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}

	if err := app.Initialize(dataDir); err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)

	// Serve static files
	r.Handle("/assets/*", http.StripPrefix("/assets/", http.FileServer(http.Dir("./ui/assets/"))))

	// Initialize handlers
	projectHandlers := handlers.NewProjectHandlers(app.GetProjectService())
	discoveryHandlers := handlers.NewDiscoveryHandlers(app.GetDiscoveryService())

	// Register routes
	projectHandlers.RegisterRoutes(r)
	discoveryHandlers.RegisterRoutes(r)

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

	slog.Info("Server starting", "address", "http://127.0.0.1:3333")
	if err := http.ListenAndServe("127.0.0.1:3333", r); err != nil {
		panic(err)
	}
}
