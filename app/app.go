// Package app provides the main application context for Oar, managing the database and services.
package app

import (
	"os"

	"github.com/oar-cd/oar/db"
	"github.com/oar-cd/oar/models"
	"github.com/oar-cd/oar/services"
	"gorm.io/gorm"
)

var (
	database         *gorm.DB
	projectService   services.ProjectManager
	discoveryService *services.ProjectDiscoveryService
	gitService       services.GitExecutor
	config           *services.Config
)

// InitializeWithConfig initializes the app with a pre-configured Config
func InitializeWithConfig(cfg *services.Config) error {
	var err error

	// Store the provided config
	config = cfg

	// Ensure required directories exist
	if err := os.MkdirAll(config.DataDir, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(config.TmpDir, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(config.WorkspaceDir, 0755); err != nil {
		return err
	}

	// Initialize database using config
	database, err = db.InitDB(config.DataDir)
	if err != nil {
		return err
	}

	// Run database migrations
	if err := models.AutoMigrateAll(database); err != nil {
		return err
	}

	gitService = services.NewGitService(config)

	// Initialize encryption service
	encryption, err := services.NewEncryptionService(config.EncryptionKey)
	if err != nil {
		return err
	}

	// Initialize repositories
	projectRepo := services.NewProjectRepository(database, encryption)
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

func GetGitService() services.GitExecutor {
	return gitService
}

// SetProjectServiceForTesting allows overriding the project service for testing purposes
func SetProjectServiceForTesting(service services.ProjectManager) {
	projectService = service
}
