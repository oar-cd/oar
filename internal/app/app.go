// Package app provides the main application context for Oar, managing the database and services.
package app

import (
	"github.com/ch00k/oar/db"
	"github.com/ch00k/oar/services"
	"gorm.io/gorm"
)

var (
	database         *gorm.DB
	projectService   services.ProjectManager
	discoveryService *services.ProjectDiscoveryService
	config           *services.Config
)

// InitializeWithConfig initializes the app with a pre-configured Config
func InitializeWithConfig(cfg *services.Config) error {
	var err error

	// Store the provided config
	config = cfg

	// Initialize database using config
	database, err = db.InitDB(config.DataDir)
	if err != nil {
		return err
	}

	gitService := services.NewGitService(config)

	// Initialize repositories
	projectRepo := services.NewProjectRepository(database)
	deploymentRepo := services.NewDeploymentRepository(database)

	// Initialize services with dependency injection
	projectService = services.NewProjectService(projectRepo, deploymentRepo, gitService, config)
	discoveryService = services.NewProjectDiscoveryService(gitService, config)
	return nil
}

func GetProjectService() services.ProjectManager {
	return projectService
}

func GetDiscoveryService() *services.ProjectDiscoveryService {
	return discoveryService
}
