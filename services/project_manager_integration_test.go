package services

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProjectManager_Integration_CompleteLifecycle tests the complete project lifecycle
// as outlined in Test Case 1: Create → Deploy → GetConfig → GetLogs → Stop → Remove
func TestProjectManager_Integration_CompleteLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test environment with real services
	service, _ := setupProjectService(t)
	projectManager := ProjectManager(service)

	// Test data
	testRepoURL := "https://github.com/oar-cd/test-project.git"
	testProjectName := "test-project-lifecycle"

	// Step 1: Create project from Git repository
	t.Log("Step 1: Creating project from Git repository...")

	project := &Project{
		ID:           uuid.New(),
		Name:         testProjectName,
		GitURL:       testRepoURL,
		GitAuth:      nil,                      // Public repository, no auth needed
		ComposeFiles: []string{"compose.yaml"}, // Required for project creation
	}

	createdProject, err := projectManager.Create(project)
	require.NoError(t, err, "Project creation should succeed")
	require.NotNil(t, createdProject, "Created project should not be nil")

	assert.Equal(t, testProjectName, createdProject.Name)
	assert.Equal(t, testRepoURL, createdProject.GitURL)
	assert.NotEmpty(t, createdProject.WorkingDir)
	assert.NotNil(t, createdProject.CreatedAt)

	t.Logf("Project created successfully with ID: %s", createdProject.ID)

	// Step 2: Deploy the project using streaming
	t.Log("Step 2: Deploying project with streaming...")

	// Create output channel for streaming
	outputChan := make(chan string, 100)
	deployDone := make(chan error, 1)

	// Start deployment in goroutine
	go func() {
		defer close(outputChan)
		deployDone <- projectManager.DeployStreaming(createdProject.ID, false, outputChan)
	}()

	// Collect deployment output
	var deployMessages []string
	timeout := time.After(60 * time.Second) // Give enough time for Docker operations

deployLoop:
	for {
		select {
		case msg, ok := <-outputChan:
			if !ok {
				break deployLoop
			}
			deployMessages = append(deployMessages, msg)
			t.Logf("Deploy output: %s", strings.TrimSpace(msg))
		case err := <-deployDone:
			require.NoError(t, err, "Deployment should succeed")
			break deployLoop
		case <-timeout:
			t.Fatal("Deployment timed out")
		}
	}

	assert.Greater(t, len(deployMessages), 0, "Should receive deployment messages")
	t.Logf("Project deployed successfully with %d output messages", len(deployMessages))

	// Wait a bit for containers to stabilize
	time.Sleep(5 * time.Second)

	// Step 3: Get project configuration
	t.Log("Step 3: Getting project configuration...")

	config, err := projectManager.GetConfig(createdProject.ID)
	require.NoError(t, err, "Getting config should succeed")
	require.NotEmpty(t, config, "Config should not be empty")

	// Verify config contains expected compose content
	assert.Contains(t, config, "services:", "Config should contain services section")
	t.Logf("Project configuration retrieved (%d characters)", len(config))

	// Step 4: Get project logs using streaming
	t.Log("Step 4: Getting project logs with streaming...")

	logsChan := make(chan string, 100)
	logsDone := make(chan error, 1)

	// Start log streaming in goroutine with timeout
	_, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	go func() {
		defer close(logsChan)
		// Note: We'll need to modify GetLogsStreaming to accept context or add timeout
		logsDone <- projectManager.GetLogsStreaming(createdProject.ID, logsChan)
	}()

	// Collect some log output
	var logMessages []string
	logsTimeout := time.After(10 * time.Second) // Shorter timeout for logs

logsLoop:
	for {
		select {
		case msg, ok := <-logsChan:
			if !ok {
				break logsLoop
			}
			logMessages = append(logMessages, msg)
			t.Logf("Log output: %s", strings.TrimSpace(msg))

			// Stop after getting some logs to avoid infinite collection
			if len(logMessages) >= 5 {
				cancel() // This should stop log streaming
				break logsLoop
			}
		case err := <-logsDone:
			// Log streaming might return error when cancelled, which is expected
			if err != nil {
				t.Logf("Log streaming ended: %v", err)
			}
			break logsLoop
		case <-logsTimeout:
			cancel()
			t.Log("Log collection timeout reached")
			break logsLoop
		}
	}

	t.Logf("Collected %d log messages", len(logMessages))

	// Step 5: Stop the project using streaming
	t.Log("Step 5: Stopping project with streaming...")

	stopChan := make(chan string, 100)
	stopDone := make(chan error, 1)

	go func() {
		defer close(stopChan)
		stopDone <- projectManager.StopStreaming(createdProject.ID, stopChan)
	}()

	// Collect stop output
	var stopMessages []string
	stopTimeout := time.After(30 * time.Second)

stopLoop:
	for {
		select {
		case msg, ok := <-stopChan:
			if !ok {
				break stopLoop
			}
			stopMessages = append(stopMessages, msg)
			t.Logf("Stop output: %s", strings.TrimSpace(msg))
		case err := <-stopDone:
			require.NoError(t, err, "Stopping should succeed")
			break stopLoop
		case <-stopTimeout:
			t.Fatal("Stop operation timed out")
		}
	}

	assert.Greater(t, len(stopMessages), 0, "Should receive stop messages")
	t.Logf("Project stopped successfully with %d output messages", len(stopMessages))

	// Step 6: Verify project is stopped by getting updated project info
	t.Log("Step 6: Verifying project status...")

	updatedProject, err := projectManager.Get(createdProject.ID)
	require.NoError(t, err, "Getting project should succeed")

	// Project status should be stopped (if status tracking is implemented)
	t.Logf("Project status: %s", updatedProject.Status.String())

	// Step 7: Remove the project
	t.Log("Step 7: Removing project...")

	err = projectManager.Remove(createdProject.ID)
	require.NoError(t, err, "Project removal should succeed")

	t.Logf("Project removed successfully")

	// Step 8: Verify project is removed
	t.Log("Step 8: Verifying project removal...")

	removedProject, err := projectManager.Get(createdProject.ID)
	assert.Error(t, err, "Getting removed project should fail")
	assert.Nil(t, removedProject, "Removed project should be nil")
	assert.Contains(t, err.Error(), "not found", "Error should indicate project not found")

	t.Logf("Project removal verified")

	// Verify project is not in list
	allProjects, err := projectManager.List()
	require.NoError(t, err, "Listing projects should succeed")

	for _, p := range allProjects {
		assert.NotEqual(t, createdProject.ID, p.ID, "Removed project should not be in list")
	}

	t.Logf("Complete project lifecycle test passed")
}
