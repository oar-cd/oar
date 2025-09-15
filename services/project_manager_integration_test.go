package services

import (
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
		GitBranch:    "main",                   // Use main branch for basic test
		GitAuth:      nil,                      // Public repository, no auth needed
		ComposeFiles: []string{"compose.yaml"}, // Basic compose file in main branch
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

	// Step 2.5: Verify project status after deployment
	t.Log("Step 2.5: Verifying project status after deployment...")

	status, err := projectManager.GetStatus(createdProject.ID)
	require.NoError(t, err, "Getting status should succeed")
	require.NotNil(t, status, "Status should not be nil")

	// Project should be running after deployment
	assert.Equal(t, "running", status.Status, "Project should be running after deployment")
	assert.NotEmpty(t, status.Uptime, "Should have uptime when running")

	// Should have exactly 2 containers: web and redis
	assert.Len(t, status.Containers, 2, "Should have exactly 2 containers (web, redis)")

	// Verify expected services are present
	serviceNames := make([]string, len(status.Containers))
	for i, container := range status.Containers {
		serviceNames[i] = container.Service
		assert.Equal(t, "running", container.State, "Container %s should be in running state", container.Service)
		assert.Contains(t, container.Name, testProjectName, "Container name should contain project name")
		assert.NotEmpty(t, container.Status, "Container should have status description")
		assert.NotEmpty(t, container.RunningFor, "Container should have running duration")
	}
	assert.Contains(t, serviceNames, "web", "Should have web service")
	assert.Contains(t, serviceNames, "redis", "Should have redis service")

	t.Logf("Project status verified: %s with uptime %s", status.Status, status.Uptime)
	t.Logf("Containers verified: %v", serviceNames)

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

	// Get logs (no streaming needed)

	go func() {
		defer close(logsChan)
		// Get static logs instead of streaming
		logs, err := projectManager.GetLogs(createdProject.ID)
		if err != nil {
			logsDone <- err
			return
		}
		// Send each line to the channel to simulate the old streaming behavior for the test
		lines := strings.Split(logs, "\n")
		for _, line := range lines {
			if line != "" {
				logsChan <- line
			}
		}
		logsDone <- nil
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
				// No need to cancel since we're using static logs
				break logsLoop
			}
		case err := <-logsDone:
			// Log streaming might return error when cancelled, which is expected
			if err != nil {
				t.Logf("Log streaming ended: %v", err)
			}
			break logsLoop
		case <-logsTimeout:
			// No timeout needed since we're using static logs
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

	// Step 6: Verify project is stopped by checking container status
	t.Log("Step 6: Verifying project status after stopping...")

	stoppedStatus, err := projectManager.GetStatus(createdProject.ID)
	require.NoError(t, err, "Getting status should succeed")
	require.NotNil(t, stoppedStatus, "Status should not be nil")

	// Project should be stopped after stopping
	assert.Equal(t, "stopped", stoppedStatus.Status, "Project should be stopped after stop operation")
	assert.Empty(t, stoppedStatus.Uptime, "Should have no uptime when stopped")

	// All containers should be stopped or removed
	for _, container := range stoppedStatus.Containers {
		assert.NotEqual(
			t,
			"running",
			container.State,
			"Container %s should not be running after stop",
			container.Service,
		)
	}

	t.Logf("Project status verified after stop: %s", stoppedStatus.Status)

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

// TestProjectManager_Integration_MergeStrategy tests the merge strategy with multiple Compose files
// using the -f flag to combine multiple compose files
func TestProjectManager_Integration_MergeStrategy(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	service, _ := setupProjectService(t)
	projectManager := ProjectManager(service)

	testRepoURL := "https://github.com/oar-cd/test-project.git"
	testProjectName := "test-project-merge-strategy"

	t.Log("Testing merge strategy with multiple compose files...")

	project := &Project{
		ID:        uuid.New(),
		Name:      testProjectName,
		GitURL:    testRepoURL,
		GitBranch: "compose-files-merge", // Use compose-files-merge branch
		GitAuth:   nil,
		// Using multiple compose files for merge strategy
		ComposeFiles: []string{"compose.yaml", "compose.override.yaml"},
	}

	createdProject, err := projectManager.Create(project)
	require.NoError(t, err, "Project creation should succeed")
	require.NotNil(t, createdProject, "Created project should not be nil")

	// Deploy the project
	outputChan := make(chan string, 100)
	deployDone := make(chan error, 1)

	go func() {
		defer close(outputChan)
		deployDone <- projectManager.DeployStreaming(createdProject.ID, false, outputChan)
	}()

	timeout := time.After(60 * time.Second)

deployLoop:
	for {
		select {
		case msg, ok := <-outputChan:
			if !ok {
				break deployLoop
			}
			t.Logf("Deploy output: %s", strings.TrimSpace(msg))
		case err := <-deployDone:
			require.NoError(t, err, "Deployment should succeed")
			break deployLoop
		case <-timeout:
			t.Fatal("Deployment timed out")
		}
	}

	// Verify project status after deployment (merge strategy should have web, redis, db)
	t.Log("Verifying merge strategy project status...")

	status, err := projectManager.GetStatus(createdProject.ID)
	require.NoError(t, err, "Getting status should succeed")
	require.NotNil(t, status, "Status should not be nil")

	// Project should be running
	assert.Equal(t, "running", status.Status, "Project should be running after deployment")

	// Should have exactly 3 containers: web, redis, db
	assert.Len(t, status.Containers, 3, "Should have exactly 3 containers (web, redis, db)")

	// Verify expected services are present
	serviceNames := make([]string, len(status.Containers))
	for i, container := range status.Containers {
		serviceNames[i] = container.Service
		assert.Equal(t, "running", container.State, "Container %s should be running", container.Service)
	}
	assert.Contains(t, serviceNames, "web", "Should have web service")
	assert.Contains(t, serviceNames, "redis", "Should have redis service")
	assert.Contains(t, serviceNames, "db", "Should have db service")

	t.Logf("Merge strategy status verified: %v", serviceNames)

	// Verify merged configuration
	config, err := projectManager.GetConfig(createdProject.ID)
	require.NoError(t, err, "Getting config should succeed")
	require.NotEmpty(t, config, "Config should not be empty")

	// Config should show merged result from both files
	assert.Contains(t, config, "services:", "Config should contain services section")
	t.Logf("Merge strategy configuration verified (%d characters)", len(config))

	// Cleanup
	err = projectManager.StopStreaming(createdProject.ID, make(chan string, 100))
	require.NoError(t, err, "Stopping should succeed")

	err = projectManager.Remove(createdProject.ID)
	require.NoError(t, err, "Project removal should succeed")

	t.Logf("Merge strategy test completed successfully")
}

// TestProjectManager_Integration_ExtendStrategy tests the extend strategy where one compose file extends another
func TestProjectManager_Integration_ExtendStrategy(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	service, _ := setupProjectService(t)
	projectManager := ProjectManager(service)

	testRepoURL := "https://github.com/oar-cd/test-project.git"
	testProjectName := "test-project-extend-strategy"

	t.Log("Testing extend strategy with compose files...")

	project := &Project{
		ID:        uuid.New(),
		Name:      testProjectName,
		GitURL:    testRepoURL,
		GitBranch: "compose-files-extend", // Use compose-files-extend branch
		GitAuth:   nil,
		// Using compose file that extends another
		ComposeFiles: []string{"compose.yaml"},
	}

	createdProject, err := projectManager.Create(project)
	require.NoError(t, err, "Project creation should succeed")
	require.NotNil(t, createdProject, "Created project should not be nil")

	// Deploy the project
	outputChan := make(chan string, 100)
	deployDone := make(chan error, 1)

	go func() {
		defer close(outputChan)
		deployDone <- projectManager.DeployStreaming(createdProject.ID, false, outputChan)
	}()

	timeout := time.After(60 * time.Second)

deployLoop:
	for {
		select {
		case msg, ok := <-outputChan:
			if !ok {
				break deployLoop
			}
			t.Logf("Deploy output: %s", strings.TrimSpace(msg))
		case err := <-deployDone:
			require.NoError(t, err, "Deployment should succeed")
			break deployLoop
		case <-timeout:
			t.Fatal("Deployment timed out")
		}
	}

	// Verify extended configuration
	config, err := projectManager.GetConfig(createdProject.ID)
	require.NoError(t, err, "Getting config should succeed")
	require.NotEmpty(t, config, "Config should not be empty")

	assert.Contains(t, config, "services:", "Config should contain services section")
	t.Logf("Extend strategy configuration verified (%d characters)", len(config))

	// Cleanup
	err = projectManager.StopStreaming(createdProject.ID, make(chan string, 100))
	require.NoError(t, err, "Stopping should succeed")

	err = projectManager.Remove(createdProject.ID)
	require.NoError(t, err, "Project removal should succeed")

	t.Logf("Extend strategy test completed successfully")
}

// TestProjectManager_Integration_IncludeStrategy tests the include strategy where compose files include others
func TestProjectManager_Integration_IncludeStrategy(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	service, _ := setupProjectService(t)
	projectManager := ProjectManager(service)

	testRepoURL := "https://github.com/oar-cd/test-project.git"
	testProjectName := "test-project-include-strategy"

	t.Log("Testing include strategy with compose files...")

	project := &Project{
		ID:        uuid.New(),
		Name:      testProjectName,
		GitURL:    testRepoURL,
		GitBranch: "compose-files-include", // Use compose-files-include branch
		GitAuth:   nil,
		// Using compose file that includes others
		ComposeFiles: []string{"compose.yaml"},
	}

	createdProject, err := projectManager.Create(project)
	require.NoError(t, err, "Project creation should succeed")
	require.NotNil(t, createdProject, "Created project should not be nil")

	// Deploy the project
	outputChan := make(chan string, 100)
	deployDone := make(chan error, 1)

	go func() {
		defer close(outputChan)
		deployDone <- projectManager.DeployStreaming(createdProject.ID, false, outputChan)
	}()

	timeout := time.After(60 * time.Second)

deployLoop:
	for {
		select {
		case msg, ok := <-outputChan:
			if !ok {
				break deployLoop
			}
			t.Logf("Deploy output: %s", strings.TrimSpace(msg))
		case err := <-deployDone:
			require.NoError(t, err, "Deployment should succeed")
			break deployLoop
		case <-timeout:
			t.Fatal("Deployment timed out")
		}
	}

	// Verify included configuration
	config, err := projectManager.GetConfig(createdProject.ID)
	require.NoError(t, err, "Getting config should succeed")
	require.NotEmpty(t, config, "Config should not be empty")

	assert.Contains(t, config, "services:", "Config should contain services section")
	t.Logf("Include strategy configuration verified (%d characters)", len(config))

	// Cleanup
	err = projectManager.StopStreaming(createdProject.ID, make(chan string, 100))
	require.NoError(t, err, "Stopping should succeed")

	err = projectManager.Remove(createdProject.ID)
	require.NoError(t, err, "Project removal should succeed")

	t.Logf("Include strategy test completed successfully")
}

// TestProjectManager_Integration_Variables tests compose file interpolation and service configuration variables
func TestProjectManager_Integration_Variables(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	service, _ := setupProjectService(t)
	projectManager := ProjectManager(service)

	testRepoURL := "https://github.com/oar-cd/test-project.git"
	testProjectName := "test-project-variables"

	t.Log("Testing compose file with variables and interpolation...")

	project := &Project{
		ID:        uuid.New(),
		Name:      testProjectName,
		GitURL:    testRepoURL,
		GitBranch: "compose-files-interpolation", // Use compose-files-interpolation branch
		GitAuth:   nil,
		// Using compose file that uses variable interpolation
		ComposeFiles: []string{"compose.yaml"},
		// Environment variables for compose file interpolation
		Variables: []string{
			"APP_VERSION=latest",
			"BACKEND_PORT=3000",
			"DEBUG_MODE=true",
			"NODE_ENV=development",
			"DB_NAME=testapp",
			"DB_USER=testuser",
			"DB_PASS=testpass",
			"CUSTOM_VALUE=test-value-123",
		},
	}

	createdProject, err := projectManager.Create(project)
	require.NoError(t, err, "Project creation should succeed")
	require.NotNil(t, createdProject, "Created project should not be nil")

	// Deploy the project
	outputChan := make(chan string, 100)
	deployDone := make(chan error, 1)

	go func() {
		defer close(outputChan)
		deployDone <- projectManager.DeployStreaming(createdProject.ID, false, outputChan)
	}()

	timeout := time.After(60 * time.Second)

deployLoop:
	for {
		select {
		case msg, ok := <-outputChan:
			if !ok {
				break deployLoop
			}
			t.Logf("Deploy output: %s", strings.TrimSpace(msg))
		case err := <-deployDone:
			require.NoError(t, err, "Deployment should succeed")
			break deployLoop
		case <-timeout:
			t.Fatal("Deployment timed out")
		}
	}

	// Verify variable interpolation worked
	config, err := projectManager.GetConfig(createdProject.ID)
	require.NoError(t, err, "Getting config should succeed")
	require.NotEmpty(t, config, "Config should not be empty")

	// Config should show interpolated values
	assert.Contains(t, config, "services:", "Config should contain services section")
	assert.Contains(t, config, "3000", "Config should contain interpolated port value")
	assert.Contains(t, config, "latest", "Config should contain interpolated version value")
	assert.Contains(t, config, "test-value-123", "Config should contain interpolated custom value")
	t.Logf("Variable interpolation verified in configuration (%d characters)", len(config))

	// Cleanup
	err = projectManager.StopStreaming(createdProject.ID, make(chan string, 100))
	require.NoError(t, err, "Stopping should succeed")

	err = projectManager.Remove(createdProject.ID)
	require.NoError(t, err, "Project removal should succeed")

	t.Logf("Variables test completed successfully")
}
