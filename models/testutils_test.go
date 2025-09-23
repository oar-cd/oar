package models

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/oar-cd/oar/db"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *gorm.DB {
	// Initialize database using shared utility (same as production)
	database, err := db.InitDatabase(db.DBConfig{
		Path:     ":memory:",
		LogLevel: logger.Silent,
	})
	require.NoError(t, err)

	// Run migrations for all models (single source of truth)
	err = AutoMigrateAll(database)
	require.NoError(t, err)

	return database
}

// createTestProjectModel creates a test project model for database layer testing
func createTestProjectModel() *ProjectModel {
	return &ProjectModel{
		BaseModel: BaseModel{
			ID: uuid.New(),
		},
		Name:         "test-project",
		GitURL:       "https://github.com/test/repo.git",
		GitBranch:    "main",
		WorkingDir:   "/tmp/test-project",
		ComposeFiles: "docker-compose.yml",
		Variables:    "KEY1=value1",
		Status:       "stopped",
		LastCommit:   stringPtr("abc123"),
	}
}

// createTestDeploymentModel creates a test deployment model for database layer testing
func createTestDeploymentModel(projectID uuid.UUID) *DeploymentModel {
	return &DeploymentModel{
		BaseModel: BaseModel{
			ID: uuid.New(),
		},
		ProjectID:  projectID,
		CommitHash: "def456",
		Status:     "success",
		Stdout:     "Container started successfully",
		Stderr:     "",
	}
}

// Utility functions
func stringPtr(s string) *string {
	return &s
}
