// Package app provides the main application context for Oar, managing the database and services.
package app

import (
	"github.com/ch00k/oar/db"
	"github.com/ch00k/oar/services"
	"gorm.io/gorm"
)

var (
	database       *gorm.DB
	projectService services.ProjectManager
	config         *services.Config
)

func Initialize(dataDir string) error {
	var err error

	// Initialize configuration
	config = services.NewConfig(dataDir)

	// Initialize database
	database, err = db.InitDB(dataDir)
	if err != nil {
		return err
	}

	// Initialize repositories
	projectRepo := services.NewProjectRepository(database)
	deploymentRepo := services.NewDeploymentRepository(database)

	// Initialize services with dependency injection
	projectService = services.NewProjectServiceWithDefaults(projectRepo, deploymentRepo, config)
	return nil
}

func GetProjectService() services.ProjectManager {
	return projectService
}
