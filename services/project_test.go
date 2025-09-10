package services

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for ProjectService.List()
func TestProjectService_List_Success(t *testing.T) {
	service, repo, _, _, _ := setupMockProjectService(t)

	// Add test projects to repository
	project1 := createTestProjectWithOptions(ProjectOptions{Name: "project-1"})
	project2 := createTestProjectWithOptions(ProjectOptions{Name: "project-2"})

	repo.projects[project1.ID] = project1
	repo.projects[project2.ID] = project2

	// Test
	projects, err := service.List()

	// Assertions
	assert.NoError(t, err)
	assert.Len(t, projects, 2)
}

func TestProjectService_List_Empty(t *testing.T) {
	service, _, _, _, _ := setupMockProjectService(t)

	// Test
	projects, err := service.List()

	// Assertions
	assert.NoError(t, err)
	assert.Empty(t, projects)
}

// Tests for ProjectService.Get()
func TestProjectService_Get_Success(t *testing.T) {
	service, repo, _, _, _ := setupMockProjectService(t)

	// Add test project
	testProject := createTestProject()
	repo.projects[testProject.ID] = testProject

	// Test
	project, err := service.Get(testProject.ID)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, testProject.ID, project.ID)
	assert.Equal(t, testProject.Name, project.Name)
}

func TestProjectService_Get_NotFound(t *testing.T) {
	service, _, _, _, _ := setupMockProjectService(t)

	// Test with non-existent ID
	nonExistentID := uuid.New()
	project, err := service.Get(nonExistentID)

	// Assertions
	assert.Error(t, err)
	assert.Nil(t, project)
	assert.Contains(t, err.Error(), "project not found")
}

// Tests for ProjectService.Create()
func TestProjectService_Create_Success(t *testing.T) {
	service, _, _, gitService, _ := setupMockProjectService(t)

	// Setup git service mock
	gitService.CloneFunc = func(gitURL string, gitAuth *GitAuthConfig, workingDir string) error {
		// Create a mock git repository
		return os.MkdirAll(filepath.Join(workingDir, ".git"), 0o755)
	}
	gitService.GetLatestCommitFunc = func(workingDir string) (string, error) {
		return "abc123", nil
	}

	// Test project
	testProject := createTestProject()

	// Test
	createdProject, err := service.Create(testProject)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, createdProject)
	assert.Equal(t, testProject.Name, createdProject.Name)
	assert.Equal(t, testProject.GitURL, createdProject.GitURL)
	assert.NotEmpty(t, createdProject.WorkingDir)
	assert.NotNil(t, createdProject.LastCommit)
	assert.Equal(t, "abc123", *createdProject.LastCommit)
	assert.NotNil(t, createdProject.CreatedAt)
	assert.NotNil(t, createdProject.UpdatedAt)
}

func TestProjectService_Create_GitCloneFails(t *testing.T) {
	service, _, _, gitService, _ := setupMockProjectService(t)

	// Setup git service to fail
	gitService.CloneFunc = func(gitURL string, gitAuth *GitAuthConfig, workingDir string) error {
		return fmt.Errorf("git clone failed: repository not found")
	}

	// Test project
	testProject := createTestProject()

	// Test
	createdProject, err := service.Create(testProject)

	// Assertions
	assert.Error(t, err)
	assert.Nil(t, createdProject)
	assert.Contains(t, err.Error(), "git clone failed")
}

func TestProjectService_Create_DuplicateName(t *testing.T) {
	service, repo, _, gitService, _ := setupMockProjectService(t)

	// Setup git service mock
	gitService.CloneFunc = func(gitURL string, gitAuth *GitAuthConfig, workingDir string) error {
		return os.MkdirAll(filepath.Join(workingDir, ".git"), 0o755)
	}
	gitService.GetLatestCommitFunc = func(workingDir string) (string, error) {
		return "abc123", nil
	}

	// Create first project
	existingProject := createTestProjectWithOptions(ProjectOptions{Name: "duplicate-name"})
	repo.projects[existingProject.ID] = existingProject

	// Test project with same name
	testProject := createTestProjectWithOptions(ProjectOptions{Name: "duplicate-name"})

	// Test
	createdProject, err := service.Create(testProject)

	// Assertions
	assert.Error(t, err)
	assert.Nil(t, createdProject)
	assert.Contains(t, err.Error(), "already exists")
}

func TestProjectService_Create_EmptyComposeFiles(t *testing.T) {
	service, _, _, _, _ := setupMockProjectService(t)

	// Test project with empty compose files
	testProject := createTestProjectWithOptions(ProjectOptions{
		Name:                 "test-project",
		ComposeFiles:         []string{}, // Empty compose files
		OverrideComposeFiles: true,       // Explicitly override to empty
	})

	// Test
	createdProject, err := service.Create(testProject)

	// Assertions
	assert.Error(t, err)
	assert.Nil(t, createdProject)
	assert.Contains(t, err.Error(), "compose files are required")
}

// Tests for ProjectService.Update()
func TestProjectService_Update_Success(t *testing.T) {
	service, repo, _, _, _ := setupMockProjectService(t)

	// Add test project to repository
	testProject := createTestProject()
	repo.projects[testProject.ID] = testProject

	// Modify project
	testProject.Name = "updated-name"
	testProject.Status = ProjectStatusRunning

	// Test
	err := service.Update(testProject)

	// Assertions
	assert.NoError(t, err)

	// Verify update in repository
	updatedProject := repo.projects[testProject.ID]
	assert.Equal(t, "updated-name", updatedProject.Name)
	assert.Equal(t, ProjectStatusRunning, updatedProject.Status)
	assert.NotNil(t, updatedProject.UpdatedAt)
}

func TestProjectService_Update_NotFound(t *testing.T) {
	service, _, _, _, _ := setupMockProjectService(t)

	// Test project not in repository
	testProject := createTestProject()

	// Test
	err := service.Update(testProject)

	// Assertions
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "project not found")
}

// Tests for ProjectStatus enum
func TestProjectStatus_String(t *testing.T) {
	tests := []struct {
		status   ProjectStatus
		expected string
	}{
		{ProjectStatusRunning, "running"},
		{ProjectStatusStopped, "stopped"},
		{ProjectStatusError, "error"},
		{ProjectStatusUnknown, "unknown"},
		{ProjectStatus(999), "unknown"}, // Invalid status
	}

	for _, test := range tests {
		t.Run(test.expected, func(t *testing.T) {
			result := test.status.String()
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestParseProjectStatus(t *testing.T) {
	tests := []struct {
		input       string
		expected    ProjectStatus
		expectError bool
	}{
		{"running", ProjectStatusRunning, false},
		{"stopped", ProjectStatusStopped, false},
		{"error", ProjectStatusError, false},
		{"unknown", ProjectStatusUnknown, false},
		{"invalid", ProjectStatusUnknown, true},
		{"", ProjectStatusUnknown, true},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result, err := ParseProjectStatus(test.input)

			if test.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid project status")
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, test.expected, result)
		})
	}
}

// Integration test for project directory structure
func TestProjectService_ProjectDirectoryStructure(t *testing.T) {
	service, _, _, gitService, tempDir := setupMockProjectService(t)

	// Setup git service mock
	gitService.CloneFunc = func(gitURL string, gitAuth *GitAuthConfig, workingDir string) error {
		// Create realistic git repository structure
		gitDir := filepath.Join(workingDir, ".git")
		err := os.MkdirAll(gitDir, 0o755)
		if err != nil {
			return err
		}

		// Create some mock files
		files := map[string]string{
			"docker-compose.yml": "version: '3'\nservices:\n  web:\n    image: nginx",
			"README.md":          "# Test Project",
			".env":               "NODE_ENV=production",
		}

		for filename, content := range files {
			err = os.WriteFile(filepath.Join(workingDir, filename), []byte(content), 0o644)
			if err != nil {
				return err
			}
		}

		return nil
	}
	gitService.GetLatestCommitFunc = func(workingDir string) (string, error) {
		return "commit-hash-123", nil
	}

	// Test project
	testProject := createTestProjectWithOptions(ProjectOptions{Name: "structure-test"})

	// Test
	createdProject, err := service.Create(testProject)
	require.NoError(t, err)

	// Verify project directory structure
	expectedWorkspaceDir := filepath.Join(tempDir, "projects")
	assert.Contains(t, createdProject.WorkingDir, expectedWorkspaceDir)

	gitDir, err := createdProject.GitDir()
	require.NoError(t, err)

	// Check that git directory exists
	_, err = os.Stat(gitDir)
	assert.NoError(t, err)

	// Check that .git subdirectory exists
	_, err = os.Stat(filepath.Join(gitDir, ".git"))
	assert.NoError(t, err)

	// Check that compose file exists
	_, err = os.Stat(filepath.Join(gitDir, "docker-compose.yml"))
	assert.NoError(t, err)
}
