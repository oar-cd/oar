package services

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/oar-cd/oar/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for ProjectRepository
func TestProjectRepository_Create_Success(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProjectRepository(db, setupTestEncryption(t))

	project := createTestProject()
	project.Name = "unique-create-project"

	// Test
	result, err := repo.Create(project)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, project.Name, result.Name)
	assert.Equal(t, project.GitURL, result.GitURL)
	assert.NotZero(t, result.CreatedAt)
	assert.NotZero(t, result.UpdatedAt)
}

func TestProjectRepository_Create_UniqueNameConstraint(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProjectRepository(db, setupTestEncryption(t))

	// Create first project
	project1 := createTestProject()
	project1.Name = "duplicate-name"
	_, err := repo.Create(project1)
	require.NoError(t, err)

	// Try to create second project with same name
	project2 := createTestProject()
	project2.Name = "duplicate-name"
	project2.ID = uuid.New() // Different ID

	// Test
	result, err := repo.Create(project2)

	// Assertions
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "UNIQUE constraint failed")
}

func TestProjectRepository_FindByID_Success(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProjectRepository(db, setupTestEncryption(t))

	// Create project
	originalProject := createTestProject()
	originalProject.Name = "findable-by-id"
	createdProject, err := repo.Create(originalProject)
	require.NoError(t, err)

	// Test
	foundProject, err := repo.FindByID(createdProject.ID)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, foundProject)
	assert.Equal(t, createdProject.ID, foundProject.ID)
	assert.Equal(t, "findable-by-id", foundProject.Name)
	assert.Equal(t, originalProject.GitURL, foundProject.GitURL)
}

func TestProjectRepository_FindByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProjectRepository(db, setupTestEncryption(t))

	// Test with non-existent ID
	nonExistentID := uuid.New()
	foundProject, err := repo.FindByID(nonExistentID)

	// Assertions
	assert.Error(t, err)
	assert.Nil(t, foundProject)
}

func TestProjectRepository_FindByName_Success(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProjectRepository(db, setupTestEncryption(t))

	// Create project
	originalProject := createTestProject()
	originalProject.Name = "findable-by-name"
	createdProject, err := repo.Create(originalProject)
	require.NoError(t, err)

	// Test
	foundProject, err := repo.FindByName("findable-by-name")

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, foundProject)
	assert.Equal(t, createdProject.ID, foundProject.ID)
	assert.Equal(t, "findable-by-name", foundProject.Name)
}

func TestProjectRepository_FindByName_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProjectRepository(db, setupTestEncryption(t))

	// Test with non-existent name
	foundProject, err := repo.FindByName("non-existent-project")

	// Assertions
	assert.Error(t, err)
	assert.Nil(t, foundProject)
}

func TestProjectRepository_Update_Success(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProjectRepository(db, setupTestEncryption(t))

	// Create project
	originalProject := createTestProject()
	originalProject.Name = "update-test-project"
	createdProject, err := repo.Create(originalProject)
	require.NoError(t, err)

	originalUpdatedAt := createdProject.UpdatedAt
	time.Sleep(1 * time.Millisecond) // Ensure different timestamp

	// Update project
	createdProject.Status = ProjectStatusRunning
	createdProject.LastCommit = stringPtr("new-commit-hash")

	// Test
	err = repo.Update(createdProject)

	// Assertions
	assert.NoError(t, err)

	// Verify update by fetching
	updatedProject, err := repo.FindByID(createdProject.ID)
	require.NoError(t, err)
	assert.Equal(t, ProjectStatusRunning, updatedProject.Status)
	assert.Equal(t, "new-commit-hash", *updatedProject.LastCommit)
	assert.True(t, updatedProject.UpdatedAt.After(originalUpdatedAt))
}

func TestProjectRepository_Delete_Success(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProjectRepository(db, setupTestEncryption(t))

	// Create project
	originalProject := createTestProject()
	originalProject.Name = "delete-test-project"
	createdProject, err := repo.Create(originalProject)
	require.NoError(t, err)

	// Test
	err = repo.Delete(createdProject.ID)

	// Assertions
	assert.NoError(t, err)

	// Verify deletion
	deletedProject, err := repo.FindByID(createdProject.ID)
	assert.Error(t, err)
	assert.Nil(t, deletedProject)
}

func TestProjectRepository_Delete_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProjectRepository(db, setupTestEncryption(t))

	// Test with non-existent ID
	nonExistentID := uuid.New()
	err := repo.Delete(nonExistentID)

	// Assertions - GORM doesn't return error for deleting non-existent records
	assert.NoError(t, err)
}

func TestProjectRepository_List_Empty(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProjectRepository(db, setupTestEncryption(t))

	// Test
	projects, err := repo.List()

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, projects)
	assert.Len(t, projects, 0)
}

func TestProjectRepository_List_Multiple(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProjectRepository(db, setupTestEncryption(t))

	// Create multiple projects
	project1 := createTestProject()
	project1.Name = "list-project-1"
	project2 := createTestProject()
	project2.Name = "list-project-2"
	project3 := createTestProject()
	project3.Name = "list-project-3"

	_, err := repo.Create(project1)
	require.NoError(t, err)
	_, err = repo.Create(project2)
	require.NoError(t, err)
	_, err = repo.Create(project3)
	require.NoError(t, err)

	// Test
	projects, err := repo.List()

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, projects)
	assert.Len(t, projects, 3)

	// Verify all projects are returned
	names := make([]string, len(projects))
	for i, p := range projects {
		names[i] = p.Name
	}
	assert.Contains(t, names, "list-project-1")
	assert.Contains(t, names, "list-project-2")
	assert.Contains(t, names, "list-project-3")
}

// Tests for DeploymentRepository
func TestDeploymentRepository_Create_Success(t *testing.T) {
	db := setupTestDB(t)
	deploymentRepo := NewDeploymentRepository(db)
	projectRepo := NewProjectRepository(db, setupTestEncryption(t))

	// Create parent project first
	project := createTestProject()
	project.Name = "deployment-parent-project"
	createdProject, err := projectRepo.Create(project)
	require.NoError(t, err)

	// Create deployment
	deployment := createTestDeployment(createdProject.ID)

	// Test
	err = deploymentRepo.Create(deployment)

	// Assertions
	assert.NoError(t, err)
}

func TestDeploymentRepository_FindByID_Success(t *testing.T) {
	db := setupTestDB(t)
	deploymentRepo := NewDeploymentRepository(db)
	projectRepo := NewProjectRepository(db, setupTestEncryption(t))

	// Create parent project first
	project := createTestProject()
	project.Name = "deployment-find-parent"
	createdProject, err := projectRepo.Create(project)
	require.NoError(t, err)

	// Create deployment
	originalDeployment := createTestDeployment(createdProject.ID)
	err = deploymentRepo.Create(originalDeployment)
	require.NoError(t, err)

	// Test
	foundDeployment, err := deploymentRepo.FindByID(originalDeployment.ID)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, foundDeployment)
	assert.Equal(t, originalDeployment.ID, foundDeployment.ID)
	assert.Equal(t, originalDeployment.ProjectID, foundDeployment.ProjectID)
	assert.Equal(t, originalDeployment.CommitHash, foundDeployment.CommitHash)
}

func TestDeploymentRepository_FindByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewDeploymentRepository(db)

	// Test with non-existent ID
	nonExistentID := uuid.New()
	foundDeployment, err := repo.FindByID(nonExistentID)

	// Assertions
	assert.Error(t, err)
	assert.Nil(t, foundDeployment)
}

func TestDeploymentRepository_ListByProjectID_Success(t *testing.T) {
	db := setupTestDB(t)
	deploymentRepo := NewDeploymentRepository(db)
	projectRepo := NewProjectRepository(db, setupTestEncryption(t))

	// Create parent project
	project := createTestProject()
	project.Name = "deployment-list-parent"
	createdProject, err := projectRepo.Create(project)
	require.NoError(t, err)

	// Create multiple deployments for the project
	deployment1 := createTestDeployment(createdProject.ID)
	deployment1.CommitHash = "commit1"
	deployment2 := createTestDeployment(createdProject.ID)
	deployment2.CommitHash = "commit2"
	deployment3 := createTestDeployment(createdProject.ID)
	deployment3.CommitHash = "commit3"

	err = deploymentRepo.Create(deployment1)
	require.NoError(t, err)
	err = deploymentRepo.Create(deployment2)
	require.NoError(t, err)
	err = deploymentRepo.Create(deployment3)
	require.NoError(t, err)

	// Create deployment for different project (should not be returned)
	otherProject := createTestProject()
	otherProject.Name = "other-project"
	otherCreatedProject, err := projectRepo.Create(otherProject)
	require.NoError(t, err)

	otherDeployment := createTestDeployment(otherCreatedProject.ID)
	err = deploymentRepo.Create(otherDeployment)
	require.NoError(t, err)

	// Test
	deployments, err := deploymentRepo.ListByProjectID(createdProject.ID)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, deployments)
	assert.Len(t, deployments, 3)

	// Verify all deployments belong to the correct project
	commitHashes := make([]string, len(deployments))
	for i, d := range deployments {
		assert.Equal(t, createdProject.ID, d.ProjectID)
		commitHashes[i] = d.CommitHash
	}
	assert.Contains(t, commitHashes, "commit1")
	assert.Contains(t, commitHashes, "commit2")
	assert.Contains(t, commitHashes, "commit3")
}

func TestDeploymentRepository_ListByProjectID_Empty(t *testing.T) {
	db := setupTestDB(t)
	deploymentRepo := NewDeploymentRepository(db)
	projectRepo := NewProjectRepository(db, setupTestEncryption(t))

	// Create project with no deployments
	project := createTestProject()
	project.Name = "empty-deployment-project"
	createdProject, err := projectRepo.Create(project)
	require.NoError(t, err)

	// Test
	deployments, err := deploymentRepo.ListByProjectID(createdProject.ID)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, deployments)
	assert.Len(t, deployments, 0)
}

// Tests for helper functions
func TestParseFiles_EmptyString(t *testing.T) {
	result := parseFiles("")
	assert.Equal(t, []string{}, result)
}

func TestParseFiles_SingleFile(t *testing.T) {
	result := parseFiles("docker-compose.yml")
	assert.Equal(t, []string{"docker-compose.yml"}, result)
}

func TestParseFiles_MultipleFiles(t *testing.T) {
	input := "docker-compose.yml\x00docker-compose.override.yml\x00docker-compose.prod.yml"
	result := parseFiles(input)
	expected := []string{"docker-compose.yml", "docker-compose.override.yml", "docker-compose.prod.yml"}
	assert.Equal(t, expected, result)
}

func TestSerializeFiles_EmptySlice(t *testing.T) {
	result := serializeFiles([]string{})
	assert.Equal(t, "", result)
}

func TestSerializeFiles_SingleFile(t *testing.T) {
	result := serializeFiles([]string{"docker-compose.yml"})
	assert.Equal(t, "docker-compose.yml", result)
}

func TestSerializeFiles_MultipleFiles(t *testing.T) {
	input := []string{"docker-compose.yml", "docker-compose.override.yml", "docker-compose.prod.yml"}
	result := serializeFiles(input)
	expected := "docker-compose.yml\x00docker-compose.override.yml\x00docker-compose.prod.yml"
	assert.Equal(t, expected, result)
}

func TestSerializeFiles_Roundtrip(t *testing.T) {
	original := []string{"file1.yml", "file2.yml", "file3.yml"}
	serialized := serializeFiles(original)
	parsed := parseFiles(serialized)
	assert.Equal(t, original, parsed)
}

// Tests for data mapping and persistence
func TestProjectRepository_FilesSerialization(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProjectRepository(db, setupTestEncryption(t))

	// Create project with multiple files
	project := createTestProject()
	project.Name = "files-serialization-test"
	project.ComposeFiles = []string{"docker-compose.yml", "docker-compose.override.yml", "docker-compose.prod.yml"}
	project.Variables = []string{"KEY1=value1", "KEY2=value2", "KEY3=value3"}

	// Create and retrieve
	createdProject, err := repo.Create(project)
	require.NoError(t, err)

	retrievedProject, err := repo.FindByID(createdProject.ID)
	require.NoError(t, err)

	// Assertions
	assert.Equal(t, project.ComposeFiles, retrievedProject.ComposeFiles)
	assert.Equal(t, project.Variables, retrievedProject.Variables)
}

func TestProjectRepository_StatusMapping(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProjectRepository(db, setupTestEncryption(t))

	tests := []struct {
		name   string
		status ProjectStatus
	}{
		{"running", ProjectStatusRunning},
		{"stopped", ProjectStatusStopped},
		{"error", ProjectStatusError},
		{"unknown", ProjectStatusUnknown},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			project := createTestProject()
			project.Name = "status-test-" + test.name
			project.Status = test.status

			// Create and retrieve
			createdProject, err := repo.Create(project)
			require.NoError(t, err)

			retrievedProject, err := repo.FindByID(createdProject.ID)
			require.NoError(t, err)

			// Assertions
			assert.Equal(t, test.status, retrievedProject.Status)
		})
	}
}

func TestDeploymentRepository_StatusMapping(t *testing.T) {
	db := setupTestDB(t)
	deploymentRepo := NewDeploymentRepository(db)
	projectRepo := NewProjectRepository(db, setupTestEncryption(t))

	// Create parent project
	project := createTestProject()
	project.Name = "deployment-status-parent"
	createdProject, err := projectRepo.Create(project)
	require.NoError(t, err)

	tests := []struct {
		name   string
		status DeploymentStatus
	}{
		{"started", DeploymentStatusStarted},
		{"completed", DeploymentStatusCompleted},
		{"failed", DeploymentStatusFailed},
		{"unknown", DeploymentStatusUnknown},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			deployment := createTestDeployment(createdProject.ID)
			deployment.Status = test.status

			// Create and retrieve
			err := deploymentRepo.Create(deployment)
			require.NoError(t, err)

			retrievedDeployment, err := deploymentRepo.FindByID(deployment.ID)
			require.NoError(t, err)

			// Assertions
			assert.Equal(t, test.status, retrievedDeployment.Status)
		})
	}
}

// Tests for edge cases and error scenarios
func TestProjectRepository_NullCommitHandling(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProjectRepository(db, setupTestEncryption(t))

	// Create project with nil LastCommit
	project := createTestProject()
	project.Name = "null-commit-test"
	project.LastCommit = nil

	// Create and retrieve
	createdProject, err := repo.Create(project)
	require.NoError(t, err)

	retrievedProject, err := repo.FindByID(createdProject.ID)
	require.NoError(t, err)

	// Assertions
	assert.Nil(t, retrievedProject.LastCommit)
}

func TestProjectRepository_InvalidStatusHandling(t *testing.T) {
	db := setupTestDB(t)

	// Create project model directly with invalid status
	invalidModel := &models.ProjectModel{
		BaseModel: models.BaseModel{
			ID: uuid.New(),
		},
		Name:         "invalid-status-test",
		GitURL:       "https://github.com/test/repo.git",
		GitBranch:    "main",
		Status:       "invalid-status",
		ComposeFiles: "docker-compose.yml", // Required by constraint
		Variables:    "",
		WorkingDir:   "/tmp/test-project",
	}

	err := db.Create(invalidModel).Error
	require.NoError(t, err)

	// Test repository mapping
	repo := NewProjectRepository(db, setupTestEncryption(t))
	retrievedProject, err := repo.FindByID(invalidModel.ID)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, retrievedProject)
	assert.Equal(t, ProjectStatusUnknown, retrievedProject.Status) // Should default to unknown
}

func TestDeploymentRepository_InvalidStatusHandling(t *testing.T) {
	db := setupTestDB(t)
	projectRepo := NewProjectRepository(db, setupTestEncryption(t))

	// Create parent project
	project := createTestProject()
	project.Name = "invalid-deployment-status-parent"
	createdProject, err := projectRepo.Create(project)
	require.NoError(t, err)

	// Create deployment model directly with invalid status
	invalidModel := &models.DeploymentModel{
		BaseModel: models.BaseModel{
			ID: uuid.New(),
		},
		ProjectID:  createdProject.ID,
		CommitHash: "test-commit",
		Status:     "invalid-status",
	}

	err = db.Create(invalidModel).Error
	require.NoError(t, err)

	// Test repository mapping
	deploymentRepo := NewDeploymentRepository(db)
	retrievedDeployment, err := deploymentRepo.FindByID(invalidModel.ID)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, retrievedDeployment)
	assert.Equal(t, DeploymentStatusUnknown, retrievedDeployment.Status) // Should default to unknown
}

// Tests for cascade delete behavior
func TestProjectRepository_Delete_CascadesBehavior(t *testing.T) {
	db := setupTestDB(t)
	projectRepo := NewProjectRepository(db, setupTestEncryption(t))
	deploymentRepo := NewDeploymentRepository(db)

	// Create parent project
	project := createTestProject()
	project.Name = "cascade-delete-test"
	createdProject, err := projectRepo.Create(project)
	require.NoError(t, err)

	// Create multiple deployments
	deployment1 := createTestDeployment(createdProject.ID)
	deployment1.CommitHash = "cascade-commit-1"
	deployment2 := createTestDeployment(createdProject.ID)
	deployment2.CommitHash = "cascade-commit-2"

	err = deploymentRepo.Create(deployment1)
	require.NoError(t, err)
	err = deploymentRepo.Create(deployment2)
	require.NoError(t, err)

	// Verify deployments exist
	deployments, err := deploymentRepo.ListByProjectID(createdProject.ID)
	require.NoError(t, err)
	assert.Len(t, deployments, 2)

	// Delete project - should cascade delete deployments
	err = projectRepo.Delete(createdProject.ID)
	require.NoError(t, err)

	// Verify project is deleted
	deletedProject, err := projectRepo.FindByID(createdProject.ID)
	assert.Error(t, err)
	assert.Nil(t, deletedProject)

	// Verify deployments are cascade deleted
	deployments, err = deploymentRepo.ListByProjectID(createdProject.ID)
	assert.NoError(t, err) // ListByProjectID should not error, just return empty
	assert.Len(t, deployments, 0, "Deployments should be cascade deleted when project is deleted")

	// Individual deployment lookups should fail
	_, err = deploymentRepo.FindByID(deployment1.ID)
	assert.Error(t, err, "Deployment1 should be cascade deleted")
	_, err = deploymentRepo.FindByID(deployment2.ID)
	assert.Error(t, err, "Deployment2 should be cascade deleted")
}

// Tests for repository constructor functions
func TestNewProjectRepository(t *testing.T) {
	db := setupTestDB(t)
	repo := NewProjectRepository(db, setupTestEncryption(t))

	// Assertions
	assert.NotNil(t, repo)
	assert.Implements(t, (*ProjectRepository)(nil), repo)
}

func TestNewDeploymentRepository(t *testing.T) {
	db := setupTestDB(t)
	repo := NewDeploymentRepository(db)

	// Assertions
	assert.NotNil(t, repo)
	assert.Implements(t, (*DeploymentRepository)(nil), repo)
}
