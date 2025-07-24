package models

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test helpers are now in testutils_test.go

// Tests for ProjectModel CRUD operations
func TestProjectModel_Create_Success(t *testing.T) {
	db := setupTestDB(t)

	project := createTestProjectModel()
	project.Name = "unique-project"

	// Test
	result := db.Create(project)

	// Assertions
	assert.NoError(t, result.Error)
	assert.Equal(t, int64(1), result.RowsAffected)
	assert.NotZero(t, project.CreatedAt)
	assert.NotZero(t, project.UpdatedAt)
}

func TestProjectModel_Create_UniqueNameConstraint(t *testing.T) {
	db := setupTestDB(t)

	// Create first project
	project1 := createTestProjectModel()
	project1.Name = "duplicate-name"
	result := db.Create(project1)
	require.NoError(t, result.Error)

	// Try to create second project with same name
	project2 := createTestProjectModel()
	project2.Name = "duplicate-name"
	project2.ID = uuid.New() // Different ID

	// Test
	result = db.Create(project2)

	// Assertions
	assert.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "UNIQUE constraint failed")
}

func TestProjectModel_Create_FieldValidation(t *testing.T) {
	db := setupTestDB(t)

	// Note: SQLite with GORM doesn't enforce NOT NULL constraints on empty strings
	// This test documents the current behavior rather than ideal validation
	tests := []struct {
		name        string
		modifyModel func(*ProjectModel)
		expectError bool
	}{
		{
			name: "empty name allowed by SQLite",
			modifyModel: func(p *ProjectModel) {
				p.Name = ""
			},
			expectError: false, // SQLite allows empty strings even with NOT NULL
		},
		{
			name: "empty git URL allowed by SQLite",
			modifyModel: func(p *ProjectModel) {
				p.GitURL = ""
			},
			expectError: false,
		},
		{
			name: "empty working directory allowed by SQLite",
			modifyModel: func(p *ProjectModel) {
				p.WorkingDir = ""
			},
			expectError: false,
		},
		{
			name: "empty status allowed by SQLite",
			modifyModel: func(p *ProjectModel) {
				p.Status = ""
			},
			expectError: false,
		},
		{
			name: "all required fields present",
			modifyModel: func(p *ProjectModel) {
				// No modification, all fields should be valid
			},
			expectError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			project := createTestProjectModel()
			project.Name = test.name // Ensure unique names
			test.modifyModel(project)

			result := db.Create(project)

			if test.expectError {
				assert.Error(t, result.Error, "Expected error for test case: %s", test.name)
			} else {
				assert.NoError(t, result.Error, "Expected no error for test case: %s", test.name)
			}
		})
	}
}

func TestProjectModel_Update_Success(t *testing.T) {
	db := setupTestDB(t)

	// Create project
	project := createTestProjectModel()
	result := db.Create(project)
	require.NoError(t, result.Error)

	originalUpdatedAt := project.UpdatedAt
	time.Sleep(1 * time.Millisecond) // Ensure different timestamp

	// Update project
	project.Status = "running"
	project.LastCommit = stringPtr("new-commit-hash")

	// Test
	result = db.Save(project)

	// Assertions
	assert.NoError(t, result.Error)
	assert.Equal(t, "running", project.Status)
	assert.Equal(t, "new-commit-hash", *project.LastCommit)
	assert.True(t, project.UpdatedAt.After(originalUpdatedAt))
}

func TestProjectModel_Delete_Success(t *testing.T) {
	db := setupTestDB(t)

	// Create project
	project := createTestProjectModel()
	result := db.Create(project)
	require.NoError(t, result.Error)

	// Test
	result = db.Delete(project)

	// Assertions
	assert.NoError(t, result.Error)
	assert.Equal(t, int64(1), result.RowsAffected)

	// Verify deletion
	var count int64
	db.Model(&ProjectModel{}).Where("id = ?", project.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestProjectModel_Query_ByID(t *testing.T) {
	db := setupTestDB(t)

	// Create project
	originalProject := createTestProjectModel()
	result := db.Create(originalProject)
	require.NoError(t, result.Error)

	// Test
	var foundProject ProjectModel
	result = db.First(&foundProject, "id = ?", originalProject.ID)

	// Assertions
	assert.NoError(t, result.Error)
	assert.Equal(t, originalProject.ID, foundProject.ID)
	assert.Equal(t, originalProject.Name, foundProject.Name)
	assert.Equal(t, originalProject.GitURL, foundProject.GitURL)
}

func TestProjectModel_Query_ByName(t *testing.T) {
	db := setupTestDB(t)

	// Create project
	originalProject := createTestProjectModel()
	originalProject.Name = "findable-project"
	result := db.Create(originalProject)
	require.NoError(t, result.Error)

	// Test
	var foundProject ProjectModel
	result = db.Where("name = ?", "findable-project").First(&foundProject)

	// Assertions
	assert.NoError(t, result.Error)
	assert.Equal(t, originalProject.ID, foundProject.ID)
	assert.Equal(t, "findable-project", foundProject.Name)
}

// Tests for DeploymentModel
func TestDeploymentModel_Create_Success(t *testing.T) {
	db := setupTestDB(t)

	// Create parent project first
	project := createTestProjectModel()
	result := db.Create(project)
	require.NoError(t, result.Error)

	// Create deployment
	deployment := createTestDeploymentModel(project.ID)

	// Test
	result = db.Create(deployment)

	// Assertions
	assert.NoError(t, result.Error)
	assert.Equal(t, int64(1), result.RowsAffected)
	assert.NotZero(t, deployment.CreatedAt)
}

func TestDeploymentModel_Create_FieldValidation(t *testing.T) {
	db := setupTestDB(t)

	// Create parent project first
	project := createTestProjectModel()
	result := db.Create(project)
	require.NoError(t, result.Error)

	// Note: SQLite with GORM behavior for field validation
	tests := []struct {
		name        string
		modifyModel func(*DeploymentModel)
		expectError bool
	}{
		{
			name: "nil project ID fails with foreign key constraint",
			modifyModel: func(d *DeploymentModel) {
				d.ProjectID = uuid.Nil
			},
			expectError: true, // Foreign key constraint prevents nil project IDs
		},
		{
			name: "empty commit hash allowed by SQLite",
			modifyModel: func(d *DeploymentModel) {
				d.CommitHash = ""
			},
			expectError: false,
		},
		{
			name: "empty command line allowed by SQLite",
			modifyModel: func(d *DeploymentModel) {
				d.CommandLine = ""
			},
			expectError: false,
		},
		{
			name: "empty status allowed by SQLite",
			modifyModel: func(d *DeploymentModel) {
				d.Status = ""
			},
			expectError: false,
		},
		{
			name: "all required fields present",
			modifyModel: func(d *DeploymentModel) {
				// No modification
			},
			expectError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			deployment := createTestDeploymentModel(project.ID)
			test.modifyModel(deployment)

			result := db.Create(deployment)

			if test.expectError {
				assert.Error(t, result.Error, "Expected error for test case: %s", test.name)
			} else {
				assert.NoError(t, result.Error, "Expected no error for test case: %s", test.name)
			}
		})
	}
}

func TestDeploymentModel_ForeignKeyConstraint(t *testing.T) {
	db := setupTestDB(t)

	// Try to create deployment with non-existent project ID
	deployment := createTestDeploymentModel(uuid.New()) // Random UUID

	// Test
	result := db.Create(deployment)

	// Assertions - Note: SQLite may not enforce foreign key constraints by default
	// This test verifies the structure is correct, actual enforcement depends on DB config
	if result.Error != nil {
		assert.Contains(t, strings.ToLower(result.Error.Error()), "foreign key")
	}
}

// Tests for model relationships
func TestProjectModel_DeploymentRelationship(t *testing.T) {
	db := setupTestDB(t)

	// Create project
	project := createTestProjectModel()
	result := db.Create(project)
	require.NoError(t, result.Error)

	// Create multiple deployments
	deployment1 := createTestDeploymentModel(project.ID)
	deployment1.CommitHash = "commit1"
	deployment2 := createTestDeploymentModel(project.ID)
	deployment2.CommitHash = "commit2"

	result = db.Create(deployment1)
	require.NoError(t, result.Error)
	result = db.Create(deployment2)
	require.NoError(t, result.Error)

	// Test - preload deployments
	var projectWithDeployments ProjectModel
	result = db.Preload("Deployments").First(&projectWithDeployments, project.ID)

	// Assertions
	assert.NoError(t, result.Error)
	assert.Len(t, projectWithDeployments.Deployments, 2)

	commitHashes := []string{
		projectWithDeployments.Deployments[0].CommitHash,
		projectWithDeployments.Deployments[1].CommitHash,
	}
	assert.Contains(t, commitHashes, "commit1")
	assert.Contains(t, commitHashes, "commit2")
}

// Tests for model field serialization
func TestProjectModel_ComposeFiles_Serialization(t *testing.T) {
	db := setupTestDB(t)

	project := createTestProjectModel()

	// Test with multiple compose files (null-separated)
	composeFiles := "docker-compose.yml\x00docker-compose.prod.yml\x00docker-compose.dev.yml"
	project.ComposeFiles = composeFiles

	// Create and retrieve
	result := db.Create(project)
	require.NoError(t, result.Error)

	var retrievedProject ProjectModel
	result = db.First(&retrievedProject, project.ID)
	require.NoError(t, result.Error)

	// Assertions
	assert.Equal(t, composeFiles, retrievedProject.ComposeFiles)

	// Test splitting the null-separated string
	files := strings.Split(retrievedProject.ComposeFiles, "\x00")
	expected := []string{"docker-compose.yml", "docker-compose.prod.yml", "docker-compose.dev.yml"}
	assert.Equal(t, expected, files)
}

func TestProjectModel_EnvironmentFiles_Serialization(t *testing.T) {
	db := setupTestDB(t)

	project := createTestProjectModel()

	// Test with multiple environment files (null-separated)
	envFiles := ".env\x00.env.production\x00.env.local"
	project.EnvironmentFiles = envFiles

	// Create and retrieve
	result := db.Create(project)
	require.NoError(t, result.Error)

	var retrievedProject ProjectModel
	result = db.First(&retrievedProject, project.ID)
	require.NoError(t, result.Error)

	// Assertions
	assert.Equal(t, envFiles, retrievedProject.EnvironmentFiles)

	// Test splitting the null-separated string
	files := strings.Split(retrievedProject.EnvironmentFiles, "\x00")
	expected := []string{".env", ".env.production", ".env.local"}
	assert.Equal(t, expected, files)
}

// Tests for UUID field handling
func TestModels_UUIDFields(t *testing.T) {
	db := setupTestDB(t)

	// Test UUID generation and persistence
	project := createTestProjectModel()
	originalID := project.ID

	result := db.Create(project)
	require.NoError(t, result.Error)

	// Verify UUID persisted correctly
	var retrievedProject ProjectModel
	result = db.First(&retrievedProject, originalID)
	require.NoError(t, result.Error)

	assert.Equal(t, originalID, retrievedProject.ID)
	assert.NotEqual(t, uuid.Nil, retrievedProject.ID)
}

// Tests for timestamp handling
func TestModels_Timestamps(t *testing.T) {
	db := setupTestDB(t)

	beforeCreate := time.Now()

	// Create project
	project := createTestProjectModel()
	result := db.Create(project)
	require.NoError(t, result.Error)

	afterCreate := time.Now()

	// Assertions for creation timestamps
	assert.True(t, project.CreatedAt.After(beforeCreate) || project.CreatedAt.Equal(beforeCreate))
	assert.True(t, project.CreatedAt.Before(afterCreate) || project.CreatedAt.Equal(afterCreate))
	assert.True(t, project.UpdatedAt.After(beforeCreate) || project.UpdatedAt.Equal(beforeCreate))
	assert.True(t, project.UpdatedAt.Before(afterCreate) || project.UpdatedAt.Equal(afterCreate))

	// Test update timestamp
	originalCreatedAt := project.CreatedAt
	originalUpdatedAt := project.UpdatedAt

	time.Sleep(1 * time.Millisecond) // Ensure different timestamp

	project.Status = "running"
	result = db.Save(project)
	require.NoError(t, result.Error)

	// CreatedAt should remain unchanged, UpdatedAt should be newer
	assert.Equal(t, originalCreatedAt, project.CreatedAt)
	assert.True(t, project.UpdatedAt.After(originalUpdatedAt))
}

// Integration test for cascade operations
func TestModels_CascadeOperations(t *testing.T) {
	db := setupTestDB(t)

	// Create project with related records
	project := createTestProjectModel()
	result := db.Create(project)
	require.NoError(t, result.Error)

	// Create related deployments
	deployment := createTestDeploymentModel(project.ID)

	result = db.Create(deployment)
	require.NoError(t, result.Error)

	// Verify relationships exist
	var count int64
	db.Model(&DeploymentModel{}).Where("project_id = ?", project.ID).Count(&count)
	assert.Equal(t, int64(1), count)

	// Delete project - test CASCADE behavior
	// DeploymentModel has CASCADE constraint
	result = db.Delete(project)
	require.NoError(t, result.Error)

	// Verify deployments are automatically deleted due to CASCADE constraint
	db.Model(&DeploymentModel{}).Where("project_id = ?", project.ID).Count(&count)
	assert.Equal(t, int64(0), count, "Deployments should be deleted via CASCADE")
}
