package services

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

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
	gitDir := filepath.Join(composeDir, "git") // This is where NewComposeProject will look

	err := os.MkdirAll(gitDir, 0o755)
	require.NoError(t, err, "Failed to create compose directory")

	// Write compose file in git directory where NewComposeProject expects it
	composeFile := filepath.Join(gitDir, "compose.yaml")
	err = os.WriteFile(composeFile, []byte(composeContent), 0o644)
	require.NoError(t, err, "Failed to write compose file")

	// Create test config
	config := &Config{
		DataDir:       tempDir,
		LogLevel:      "warn", // Reduce noise in tests
		ColorEnabled:  false,
		DockerCommand: "docker",
		DockerHost:    "unix:///var/run/docker.sock",
		GitTimeout:    30 * time.Second,
	}

	// Create domain project for testing
	domainProject := &Project{
		ID:           uuid.New(),
		Name:         projectName,
		WorkingDir:   composeDir,
		ComposeFiles: []string{"compose.yaml"},
		Variables:    []string{},
		Status:       ProjectStatusStopped,
	}

	// Create cache directory that the preprocessing logic expects
	cacheDir, err := domainProject.CacheDir()
	require.NoError(t, err, "Failed to get cache directory path")
	err = os.MkdirAll(cacheDir, 0o755)
	require.NoError(t, err, "Failed to create cache directory")

	// Create compose project using proper constructor
	composeProject := NewComposeProject(domainProject, config)

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
	_, err := itp.Project.Down()
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

		if status.Status == "running" && len(status.Containers) > 0 {
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
	output, err := testProject.Project.Up()
	require.NoError(t, err, "Up should succeed")
	t.Logf("Up output: %s", output)

	// Wait for services to be running
	t.Log("Waiting for services to start...")
	err = testProject.WaitForServices(30 * time.Second)
	require.NoError(t, err, "Services should start within timeout")

	// Verify services are running
	t.Log("Checking project status...")
	status, err := testProject.Project.Status()
	require.NoError(t, err, "Status check should succeed")
	assert.NotNil(t, status, "Status should not be nil")
	assert.Equal(t, "running", status.Status, "Service should be running")
	assert.Greater(t, len(status.Containers), 0, "Should have running containers")

	// Test Down
	t.Log("Testing Docker Compose Down...")
	output, err = testProject.Project.Down()
	require.NoError(t, err, "Down should succeed")
	t.Logf("Down output: %s", output)
}

// TestComposeProject_Integration_MultiService tests with multiple long-running services
func TestComposeProject_Integration_MultiService(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testProject := NewIntegrationTestProject(t, GetWebServiceComposeContent())

	// Test Up
	t.Log("Starting multi-service project...")
	output, err := testProject.Project.Up()
	require.NoError(t, err, "Up should succeed")
	t.Logf("Up output: %s", output)

	// Wait for services to be ready
	err = testProject.WaitForServices(30 * time.Second)
	require.NoError(t, err, "Services should start within timeout")

	// Test Status
	t.Log("Checking service status...")
	status, err := testProject.Project.Status()
	require.NoError(t, err, "Status check should succeed")

	// Should have both web and redis services
	assert.GreaterOrEqual(t, len(status.Containers), 2, "Should have at least 2 containers")
	assert.Equal(t, "running", status.Status, "All services should be running")

	// Test Logs
	t.Log("Getting logs...")
	// Note: For integration tests, we might want to limit log retrieval
	// since logs() follows by default
	config, err := testProject.Project.GetConfig()
	require.NoError(t, err, "Config should be retrievable")
	assert.Contains(t, config, "nginx", "Config should contain nginx service")
	assert.Contains(t, config, "redis", "Config should contain redis service")

	// Test Down
	t.Log("Stopping services...")
	output, err = testProject.Project.Down()
	require.NoError(t, err, "Down should succeed")
	t.Logf("Down output: %s", output)

	// Verify services are stopped
	status, err = testProject.Project.Status()
	require.NoError(t, err, "Status check should succeed even after down")
	assert.Equal(t, "stopped", status.Status, "All services should be stopped")
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
	output, err := testProject.Project.Up()
	assert.Error(t, err, "Up should fail with invalid image")
	t.Logf("Error output: %s", output)
}

// TestComposeProject_Integration_Streaming tests real-time streaming functionality
func TestComposeProject_Integration_Streaming(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testProject := NewIntegrationTestProject(t, GetWebServiceComposeContent())

	// Test streaming Up
	t.Log("Testing streaming Up...")
	outputChan := make(chan string, 100)

	// Start streaming in goroutine
	done := make(chan error, 1)
	go func() {
		done <- testProject.Project.UpStreaming(outputChan)
	}()

	// Collect output for a reasonable time
	var messages []string
	timeout := time.After(30 * time.Second)

collectLoop:
	for {
		select {
		case msg := <-outputChan:
			messages = append(messages, msg)
			t.Logf("Received streaming message: %s", msg)
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
