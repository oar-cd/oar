package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	gitService.CloneFunc = func(gitURL string, gitBranch string, gitAuth *GitAuthConfig, workingDir string) error {
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
	assert.Equal(t, testProject.GitBranch, createdProject.GitBranch)
	assert.NotEmpty(t, createdProject.WorkingDir)
	assert.NotNil(t, createdProject.LastCommit)
	assert.Equal(t, "abc123", *createdProject.LastCommit)
	assert.NotNil(t, createdProject.CreatedAt)
	assert.NotNil(t, createdProject.UpdatedAt)
}

func TestProjectService_Create_GitCloneFails(t *testing.T) {
	service, _, _, gitService, _ := setupMockProjectService(t)

	// Setup git service to fail
	gitService.CloneFunc = func(gitURL string, gitBranch string, gitAuth *GitAuthConfig, workingDir string) error {
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
	gitService.CloneFunc = func(gitURL string, gitBranch string, gitAuth *GitAuthConfig, workingDir string) error {
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

func TestProjectService_Create_EmptyName(t *testing.T) {
	service, _, _, _, _ := setupMockProjectService(t)

	// Test project with empty name
	testProject := createTestProject()
	testProject.Name = ""

	// Test
	createdProject, err := service.Create(testProject)

	// Assertions
	assert.Error(t, err)
	assert.Nil(t, createdProject)
	assert.Contains(t, err.Error(), "name is required")
}

func TestProjectService_Create_WhitespaceOnlyName(t *testing.T) {
	service, _, _, _, _ := setupMockProjectService(t)

	// Test project with whitespace-only name
	testProject := createTestProject()
	testProject.Name = "   \t\n   "

	// Test
	createdProject, err := service.Create(testProject)

	// Assertions
	assert.Error(t, err)
	assert.Nil(t, createdProject)
	assert.Contains(t, err.Error(), "name is required")
}

func TestProjectService_Create_EmptyGitURL(t *testing.T) {
	service, _, _, _, _ := setupMockProjectService(t)

	// Test project with empty git URL
	testProject := createTestProject()
	testProject.GitURL = ""

	// Test
	createdProject, err := service.Create(testProject)

	// Assertions
	assert.Error(t, err)
	assert.Nil(t, createdProject)
	assert.Contains(t, err.Error(), "git URL is required")
}

func TestProjectService_Create_WhitespaceOnlyGitURL(t *testing.T) {
	service, _, _, _, _ := setupMockProjectService(t)

	// Test project with whitespace-only git URL
	testProject := createTestProject()
	testProject.GitURL = "   \t\n   "

	// Test
	createdProject, err := service.Create(testProject)

	// Assertions
	assert.Error(t, err)
	assert.Nil(t, createdProject)
	assert.Contains(t, err.Error(), "git URL is required")
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

func TestProjectService_Update_EmptyName(t *testing.T) {
	service, repo, _, _, _ := setupMockProjectService(t)

	// Add test project to repository
	testProject := createTestProject()
	repo.projects[testProject.ID] = testProject

	// Modify project with empty name
	testProject.Name = ""

	// Test
	err := service.Update(testProject)

	// Assertions
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestProjectService_Update_WhitespaceOnlyName(t *testing.T) {
	service, repo, _, _, _ := setupMockProjectService(t)

	// Add test project to repository
	testProject := createTestProject()
	repo.projects[testProject.ID] = testProject

	// Modify project with whitespace-only name
	testProject.Name = "   \t\n   "

	// Test
	err := service.Update(testProject)

	// Assertions
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestProjectService_Update_EmptyGitURL(t *testing.T) {
	service, repo, _, _, _ := setupMockProjectService(t)

	// Add test project to repository
	testProject := createTestProject()
	repo.projects[testProject.ID] = testProject

	// Modify project with empty git URL
	testProject.GitURL = ""

	// Test
	err := service.Update(testProject)

	// Assertions
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "git URL is required")
}

func TestProjectService_Update_EmptyComposeFiles(t *testing.T) {
	service, repo, _, _, _ := setupMockProjectService(t)

	// Add test project to repository
	testProject := createTestProject()
	repo.projects[testProject.ID] = testProject

	// Modify project with empty compose files
	testProject.ComposeFiles = []string{}

	// Test
	err := service.Update(testProject)

	// Assertions
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "compose files are required")
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
	gitService.CloneFunc = func(gitURL string, gitBranch string, gitAuth *GitAuthConfig, workingDir string) error {
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

func TestProjectService_Create_WithBranch(t *testing.T) {
	service, _, _, gitService, _ := setupMockProjectService(t)

	// Setup git service mock to verify branch parameter is passed correctly
	var receivedBranch string
	gitService.CloneFunc = func(gitURL string, gitBranch string, gitAuth *GitAuthConfig, workingDir string) error {
		receivedBranch = gitBranch
		return os.MkdirAll(filepath.Join(workingDir, ".git"), 0o755)
	}
	gitService.GetLatestCommitFunc = func(workingDir string) (string, error) {
		return "abc123", nil
	}

	// Create a test project with a specific branch
	testProject := createTestProject()
	testProject.GitBranch = "feature/test-branch"

	// Test
	createdProject, err := service.Create(testProject)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, createdProject)

	// Verify branch was passed to git service
	assert.Equal(t, "feature/test-branch", receivedBranch, "GitService.Clone should receive the correct branch")

	// Verify project has correct branch
	assert.Equal(t, "feature/test-branch", createdProject.GitBranch)
}

func TestProjectService_Create_WithEmptyBranch(t *testing.T) {
	service, _, _, gitService, _ := setupMockProjectService(t)

	// Setup git service mock to return default branch and verify detected branch is used
	var receivedBranch string
	gitService.GetDefaultBranchFunc = func(gitURL string, gitAuth *GitAuthConfig) (string, error) {
		return "main", nil // Mock default branch detection
	}
	gitService.CloneFunc = func(gitURL string, gitBranch string, gitAuth *GitAuthConfig, workingDir string) error {
		receivedBranch = gitBranch
		return os.MkdirAll(filepath.Join(workingDir, ".git"), 0o755)
	}
	gitService.GetLatestCommitFunc = func(workingDir string) (string, error) {
		return "abc123", nil
	}

	// Create a test project with empty branch (should detect and use default)
	testProject := createTestProject()
	testProject.GitBranch = ""

	// Test
	createdProject, err := service.Create(testProject)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, createdProject)

	// Verify detected default branch was passed to git service
	assert.Equal(t, "main", receivedBranch, "GitService.Clone should receive detected default branch")

	// Verify project has detected default branch
	assert.Equal(t, "main", createdProject.GitBranch)
}

// Test that DeployStreaming displays old and new commit hashes when pull=true
func TestProjectService_DeployStreaming_CommitHashDisplay(t *testing.T) {
	service, repo, deploymentRepo, gitService, _ := setupMockProjectService(t)

	// Create a test project
	testProject := createTestProject()
	repo.projects[testProject.ID] = testProject

	// Setup git service mock to simulate commit hash changes during deployment
	callCount := 0
	gitService.GetLatestCommitFunc = func(workingDir string) (string, error) {
		callCount++
		switch callCount {
		case 1:
			// First call - before pull (in prepareDeployment)
			return "abc123def456789012345678901234567890abcd", nil
		case 2:
			// Second call - before pull (in DeployStreaming)
			return "abc123def456789012345678901234567890abcd", nil
		default:
			// Third call - after pull (in DeployStreaming)
			return "xyz789def012345678901234567890abcdef01234", nil
		}
	}

	gitService.PullFunc = func(gitBranch string, gitAuth *GitAuthConfig, workingDir string) error {
		// Simulate successful pull
		return nil
	}

	// Create output channel to capture streaming messages
	outputChan := make(chan string, 100)
	done := make(chan bool)

	var messages []string
	go func() {
		for msg := range outputChan {
			messages = append(messages, msg)
		}
		done <- true
	}()

	// Test deployment with pull=true to trigger commit hash display
	err := service.DeployStreaming(testProject.ID, true, outputChan)
	close(outputChan)
	<-done

	// Assertions
	// The deployment will fail at Docker Compose stage, but we should have captured
	// the git pull messages including commit hash display before that failure
	assert.Error(t, err, "Expected error due to Docker Compose not being available")
	assert.NotEmpty(t, messages)

	// Find the commit hash message
	var commitHashMessage string
	for _, msg := range messages {
		if strings.Contains(msg, "Git pull completed successfully") &&
			strings.Contains(msg, "from") &&
			strings.Contains(msg, "to") {
			commitHashMessage = msg
			break
		}
	}

	// Verify commit hash message exists and contains expected format
	assert.NotEmpty(t, commitHashMessage, "Should contain a git pull completion message with commit hashes")

	// The message should contain shortened commit hashes (first 8 characters) showing the transition
	assert.Contains(t, commitHashMessage, "abc123de", "Should contain old commit hash (first 8 chars)")
	assert.Contains(t, commitHashMessage, "xyz789de", "Should contain new commit hash (first 8 chars)")
	assert.Contains(t, commitHashMessage, "from abc123de to xyz789de", "Should show commit hash transition")

	// Verify deployment was created
	assert.Len(t, deploymentRepo.deployments, 1)
}
