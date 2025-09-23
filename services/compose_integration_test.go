package services

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// IntegrationTestProject represents a test project for Docker Compose integration tests
type IntegrationTestProject struct {
	Name       string
	TempDir    string
	ComposeDir string
	Config     *Config
	Project    *ComposeProject
	t          *testing.T
}

// NewIntegrationTestProject creates a new integration test project with proper isolation
func NewIntegrationTestProject(t *testing.T, composeContent string) *IntegrationTestProject {
	// Create unique project name to avoid conflicts
	projectName := fmt.Sprintf("oar-test-%s", uuid.New().String()[:8])

	// Create temporary directory
	tempDir := t.TempDir()
	composeDir := filepath.Join(tempDir, "compose_project")

	err := os.MkdirAll(composeDir, 0o755)
	require.NoError(t, err, "Failed to create compose directory")

	// Write compose file
	composeFile := filepath.Join(composeDir, "compose.yaml")
	err = os.WriteFile(composeFile, []byte(composeContent), 0o644)
	require.NoError(t, err, "Failed to write compose file")

	// Create test config
	config := &Config{
		DataDir:    tempDir,
		LogLevel:   "warn", // Reduce noise in tests
		GitTimeout: 30 * time.Second,
	}

	// Create compose project
	composeProject := &ComposeProject{
		Name:         projectName,
		WorkingDir:   composeDir,
		ComposeFiles: []string{"compose.yaml"},
		Variables:    []string{},
		Config:       config,
	}

	testProject := &IntegrationTestProject{
		Name:       projectName,
		TempDir:    tempDir,
		ComposeDir: composeDir,
		Config:     config,
		Project:    composeProject,
		t:          t,
	}

	// Ensure cleanup on test completion
	t.Cleanup(func() {
		testProject.Cleanup()
	})

	return testProject
}

// Cleanup ensures all Docker resources are cleaned up
func (itp *IntegrationTestProject) Cleanup() {
	if itp.Project == nil {
		return
	}

	// Try to stop and remove all containers for this project
	_, _, err := itp.Project.Down(true)
	if err != nil {
		itp.t.Logf("Warning: Failed to clean up Docker containers for project %s: %v", itp.Name, err)
	}

	// Additional cleanup - force remove any remaining containers with this project name
	// This is a safety net in case Down() doesn't work properly
	// Note: We don't check for errors here as containers might not exist
	// TODO: Use Docker API to force remove containers if needed
	itp.t.Logf("Cleaned up integration test project: %s", itp.Name)
}

// WaitForServices waits for all services to be in running state
func (itp *IntegrationTestProject) WaitForServices(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		status, err := itp.Project.Status()
		if err != nil {
			return fmt.Errorf("failed to get status: %w", err)
		}

		if status.Status == ComposeProjectStatusRunning && len(status.Containers) > 0 {
			// All containers are running
			return nil
		}

		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("services did not start within %v", timeout)
}

// GetSimpleComposeContent returns a simple compose file for basic testing
func GetSimpleComposeContent() string {
	return `services:
  web:
    image: nginx:alpine
    restart: "no"
    ports:
      - "127.0.0.1:0:80"
`
}

// GetWebServiceComposeContent returns a compose file with a web service for testing
func GetWebServiceComposeContent() string {
	return `services:
  web:
    image: nginx:alpine
    restart: "no"
    ports:
      - "127.0.0.1:0:80"  # Bind to localhost only, use dynamic port to avoid conflicts
    environment:
      - NGINX_PORT=80

  # Add a lightweight database service for multi-service testing
  redis:
    image: redis:alpine
    restart: "no"
`
}

// GetMultiServiceComposeContent returns a compose file with multiple services
func GetMultiServiceComposeContent() string {
	return `services:
  web:
    image: nginx:alpine
    restart: "no"
    ports:
      - "127.0.0.1:0:80"
    depends_on:
      - redis

  redis:
    image: redis:alpine
    restart: "no"

  worker:
    image: alpine:latest
    restart: "no"
    command: sh -c "echo 'Worker started'; sleep 2; echo 'Worker complete'"
`
}

// Integration test helpers

// TestComposeProject_Integration_BasicLifecycle tests basic up/down lifecycle with real Docker
func TestComposeProject_Integration_BasicLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testProject := NewIntegrationTestProject(t, GetSimpleComposeContent())

	// Test Up
	t.Log("Testing Docker Compose Up...")
	stdout, stderr, err := testProject.Project.Up(true)
	require.NoError(t, err, "Up should succeed")
	t.Logf("Up stdout: %s, stderr: %s", stdout, stderr)

	// Wait for services to be running
	t.Log("Waiting for services to start...")
	err = testProject.WaitForServices(30 * time.Second)
	require.NoError(t, err, "Services should start within timeout")

	// Verify services are running
	t.Log("Checking project status...")
	status, err := testProject.Project.Status()
	require.NoError(t, err, "Status check should succeed")
	assert.NotNil(t, status, "Status should not be nil")
	assert.Equal(t, ComposeProjectStatusRunning, status.Status, "Service should be running")
	assert.Greater(t, len(status.Containers), 0, "Should have running containers")

	// Test Down
	t.Log("Testing Docker Compose Down...")
	stdout, stderr, err = testProject.Project.Down(true)
	require.NoError(t, err, "Down should succeed")
	t.Logf("Down stdout: %s, stderr: %s", stdout, stderr)
}

// TestComposeProject_Integration_MultiService tests with multiple long-running services
func TestComposeProject_Integration_MultiService(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testProject := NewIntegrationTestProject(t, GetWebServiceComposeContent())

	// Test Up
	t.Log("Starting multi-service project...")
	stdout, stderr, err := testProject.Project.Up(true)
	require.NoError(t, err, "Up should succeed")
	t.Logf("Up stdout: %s, stderr: %s", stdout, stderr)

	// Wait for services to be ready
	err = testProject.WaitForServices(30 * time.Second)
	require.NoError(t, err, "Services should start within timeout")

	// Test Status
	t.Log("Checking service status...")
	status, err := testProject.Project.Status()
	require.NoError(t, err, "Status check should succeed")

	// Should have both web and redis services
	assert.GreaterOrEqual(t, len(status.Containers), 2, "Should have at least 2 containers")
	assert.Equal(t, ComposeProjectStatusRunning, status.Status, "All services should be running")

	// Test Logs
	t.Log("Getting logs...")
	// Note: For integration tests, we might want to limit log retrieval
	// since logs() follows by default
	config, _, err := testProject.Project.GetConfig()
	require.NoError(t, err, "Config should be retrievable")
	assert.Contains(t, config, "nginx", "Config should contain nginx service")
	assert.Contains(t, config, "redis", "Config should contain redis service")

	// Test Down
	t.Log("Stopping services...")
	stdout, stderr, err = testProject.Project.Down(true)
	require.NoError(t, err, "Down should succeed")
	t.Logf("Down stdout: %s, stderr: %s", stdout, stderr)

	// Verify services are stopped
	status, err = testProject.Project.Status()
	require.NoError(t, err, "Status check should succeed even after down")
	assert.Equal(t, ComposeProjectStatusStopped, status.Status, "All services should be stopped")
}

// TestComposeProject_Integration_ErrorHandling tests error scenarios with real Docker
func TestComposeProject_Integration_ErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test with invalid compose file
	invalidCompose := `services:
  invalid:
    image: this-image-definitely-does-not-exist-12345
    restart: "no"
`

	testProject := NewIntegrationTestProject(t, invalidCompose)

	// Up should fail with invalid image
	t.Log("Testing error handling with invalid image...")
	stdout, stderr, err := testProject.Project.Up(true)
	assert.Error(t, err, "Up should fail with invalid image")
	t.Logf("Error stdout: %s, stderr: %s", stdout, stderr)
}

// TestComposeProject_Integration_Streaming tests real-time streaming functionality
func TestComposeProject_Integration_Streaming(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testProject := NewIntegrationTestProject(t, GetWebServiceComposeContent())

	// Test streaming Up
	t.Log("Testing streaming Up...")
	outputChan := make(chan StreamMessage, 100)

	// Start streaming in goroutine
	done := make(chan error, 1)
	go func() {
		done <- testProject.Project.UpStreaming(true, outputChan)
	}()

	// Collect output for a reasonable time
	var messages []string
	timeout := time.After(30 * time.Second)

collectLoop:
	for {
		select {
		case msg := <-outputChan:
			messages = append(messages, msg.Content)
			t.Logf("Received streaming message [%s]: %s", msg.Type, msg.Content)
		case err := <-done:
			require.NoError(t, err, "UpStreaming should succeed")
			break collectLoop
		case <-timeout:
			t.Fatal("UpStreaming timed out")
		}
	}

	close(outputChan)

	// Should have received some messages
	assert.Greater(t, len(messages), 0, "Should have received streaming messages")

	// Verify services are running
	err := testProject.WaitForServices(10 * time.Second)
	require.NoError(t, err, "Services should be running after streaming up")
}

// TestComposeProject_Integration_FailedDeploymentCapturesOutput tests that when a Docker Compose
// deployment fails, stdout and stderr are still captured and stored in the database.
func TestComposeProject_Integration_FailedDeploymentCapturesOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create test project with invalid compose file that will fail
	testDir := t.TempDir()
	projectName := "test-failed-deployment"
	workingDir := filepath.Join(testDir, "project")
	gitDir := filepath.Join(workingDir, "git") // Git repo goes in workingDir/git
	err := os.MkdirAll(gitDir, 0755)
	require.NoError(t, err)

	// Initialize git repository in the git directory
	repo, err := git.PlainInit(gitDir, false)
	require.NoError(t, err)

	// Create an invalid docker-compose.yml that will cause deployment to fail
	composeContent := `version: '3.8'
services:
  invalid-service:
    image: "nonexistent-image-that-should-fail-12345"
    ports:
      - "8080:80"
    command: ["echo", "This should fail during pull/creation"]
`
	composeFile := filepath.Join(gitDir, "docker-compose.yml")
	err = os.WriteFile(composeFile, []byte(composeContent), 0644)
	require.NoError(t, err)

	// Add and commit the compose file to git
	worktree, err := repo.Worktree()
	require.NoError(t, err)

	_, err = worktree.Add("docker-compose.yml")
	require.NoError(t, err)

	_, err = worktree.Commit("Add invalid compose file", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	require.NoError(t, err)

	// Set up test database and repositories
	testDB := setupTestDB(t)
	encryption := setupTestEncryption(t)
	projectRepo := NewProjectRepository(testDB, encryption)
	deploymentRepo := NewDeploymentRepository(testDB)

	// Create test project in database
	project := &Project{
		ID:           uuid.New(),
		Name:         projectName,
		GitURL:       "https://github.com/test/repo.git",
		GitBranch:    "master",   // Default branch for new repo
		WorkingDir:   workingDir, // Working directory (git repo will be in workingDir/git)
		ComposeFiles: []string{"docker-compose.yml"},
		Status:       ProjectStatusStopped,
		Variables:    []string{},
	}
	createdProject, err := projectRepo.Create(project)
	require.NoError(t, err)
	project = createdProject // Use the created project with updated fields

	// Create project service
	config := &Config{LogLevel: "debug"}
	projectService := &ProjectService{
		projectRepository:    projectRepo,
		deploymentRepository: deploymentRepo,
		gitService:           NewGitService(config),
		config:               config,
	}

	// Create output channel to capture streaming messages
	outputChan := make(chan StreamMessage, 100)
	var capturedMessages []string

	// Start goroutine to collect streaming output
	done := make(chan error, 1)
	go func() {
		defer func() { done <- nil }()
		for msg := range outputChan {
			capturedMessages = append(capturedMessages, msg.Content)
			t.Logf("Captured message [%s]: %s", msg.Type, msg.Content)
		}
	}()

	// Attempt deployment - this should fail but capture output
	deployErr := projectService.DeployStreaming(project.ID, false, outputChan)
	close(outputChan)

	// Wait for message collection to complete
	<-done

	// Deployment should fail
	require.Error(t, deployErr, "Deployment should fail with invalid Docker image")

	// Verify that we captured some streaming messages during the failed deployment
	assert.Greater(t, len(capturedMessages), 0, "Should have captured streaming messages even though deployment failed")

	// Get the deployment record from database
	deployments, err := deploymentRepo.ListByProjectID(project.ID)
	require.NoError(t, err)
	require.Len(t, deployments, 1, "Should have one deployment record")

	deployment := deployments[0]

	// Verify deployment is marked as failed
	assert.Equal(t, DeploymentStatusFailed, deployment.Status, "Deployment should be marked as failed")

	// CRITICAL TEST: Verify that stdout and/or stderr were captured despite the failure
	hasOutput := len(deployment.Stdout) > 0 || len(deployment.Stderr) > 0
	assert.True(t, hasOutput, "Failed deployment should have captured stdout and/or stderr output")

	if len(deployment.Stdout) > 0 {
		t.Logf("Captured stdout (%d chars): %s", len(deployment.Stdout), deployment.Stdout)
	}
	if len(deployment.Stderr) > 0 {
		t.Logf("Captured stderr (%d chars): %s", len(deployment.Stderr), deployment.Stderr)
	}

	// Verify that the captured output makes sense (should contain Docker-related error messages)
	combinedOutput := deployment.Stdout + deployment.Stderr
	assert.Contains(t, combinedOutput, "nonexistent-image-that-should-fail-12345",
		"Output should contain reference to the failing image")

	t.Logf("Successfully verified that failed deployment captured %d chars of output", len(combinedOutput))
}
