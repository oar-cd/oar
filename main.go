package main

import (
	"log"
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

	// Register project routes
	projectHandlers.RegisterRoutes(r)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		projectService := app.GetProjectService()
		projects, err := projectService.List()
		if err != nil {
			log.Printf("Failed to list projects: %v", err)
			projects = []*services.Project{} // Empty slice on error
		}

		component := pages.Home(projects)
		if err := component.Render(r.Context(), w); err != nil {
			http.Error(w, "Failed to render home page", http.StatusInternalServerError)
		}
	})

	log.Printf("Server starting on http://127.0.0.1:3333")
	if err := http.ListenAndServe("127.0.0.1:3333", r); err != nil {
		panic(err)
	}
}
