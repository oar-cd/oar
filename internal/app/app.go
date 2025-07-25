// Package app provides the main application context for Oar, managing the database and services.
package app

import (
	"github.com/ch00k/oar/db"
	"github.com/ch00k/oar/logging"
	"github.com/ch00k/oar/services"
	"gorm.io/gorm"
)

var (
	database         *gorm.DB
	projectService   services.ProjectManager
	discoveryService *services.ProjectDiscoveryService
	config           *services.Config
)

func Initialize(dataDir string) error {
	var err error

	logging.InitLogging()

	// Initialize configuration
	config = services.NewConfig(dataDir)

	// Initialize database
	database, err = db.InitDB(dataDir)
	if err != nil {
		return err
	}

	gitService := services.NewGitService()

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
