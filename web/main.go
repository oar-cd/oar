package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/oar-cd/oar/internal/app"
	"github.com/oar-cd/oar/logging"
	"github.com/oar-cd/oar/services"
	"github.com/oar-cd/oar/web/routes"
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
	r.Handle("/assets/*", http.StripPrefix("/assets/", http.FileServer(http.Dir("./web/assets/"))))

	// Register all routes
	routes.RegisterHomeRoutes(r)
	routes.RegisterProjectRoutes(r)
	routes.RegisterUtilityRoutes(r)

	// Start server
	address := fmt.Sprintf("%s:%d", config.HTTPHost, config.HTTPPort)
	log.Printf("Server starting on http://%s", address)

	if err := http.ListenAndServe(address, r); err != nil {
		panic(err)
	}
}
