// Package app provides the main application context for Oar, managing the database and services.
package app

import (
	"os"

	"github.com/oar-cd/oar/config"
	"github.com/oar-cd/oar/db"
	"github.com/oar-cd/oar/encryption"
	"github.com/oar-cd/oar/git"
	"github.com/oar-cd/oar/project"
	"github.com/oar-cd/oar/repository"
	"gorm.io/gorm"
)

var (
	// Version is set at build time via -ldflags
	Version = "dev"

	database       *gorm.DB
	projectService project.ProjectManager
	gitService     *git.GitService
	appConfig      *config.Config
)

// InitializeWithConfig initializes the app with a pre-configured Config
func InitializeWithConfig(cfg *config.Config) error {
	var err error

	// Store the provided config
	appConfig = cfg

	// Ensure required directories exist
	if err := os.MkdirAll(appConfig.DataDir, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(appConfig.TmpDir, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(appConfig.WorkspaceDir, 0755); err != nil {
		return err
	}

	// Initialize database using config
	database, err = db.InitDB(appConfig.DataDir)
	if err != nil {
		return err
	}

	// Run database migrations
	if err := db.AutoMigrateAll(database); err != nil {
		return err
	}

	gitService = git.NewGitService(appConfig)

	// Initialize encryption service
	encryptionSvc, err := encryption.NewEncryptionService(appConfig.EncryptionKey)
	if err != nil {
		return err
	}

	// Initialize repositories
	projectRepo := repository.NewProjectRepository(database, encryptionSvc)
	deploymentRepo := repository.NewDeploymentRepository(database)

	// Initialize services with dependency injection
	projectService = project.NewProjectService(projectRepo, deploymentRepo, gitService, appConfig)
	return nil
}

func GetProjectService() project.ProjectManager {
	return projectService
}

func GetGitService() *git.GitService {
	return gitService
}

// SetProjectServiceForTesting allows overriding the project service for testing purposes
func SetProjectServiceForTesting(service project.ProjectManager) {
	projectService = service
}
