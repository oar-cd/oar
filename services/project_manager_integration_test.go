package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
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
	outputChan := make(chan StreamMessage, 100)
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
			deployMessages = append(deployMessages, msg.Content)
			t.Logf("Deploy output [%s]: %s", msg.Type, strings.TrimSpace(msg.Content))
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

	// IMMEDIATELY capture volumes after deployment (regardless of outcome)
	volumes := getProjectNamedVolumes(t, createdProject)
	if len(volumes) > 0 {
		t.Logf("Captured %d named volumes after deployment: %v", len(volumes), volumes)
	} else {
		t.Logf("No named volumes found after deployment")
	}

	// Update cleanup to use captured volumes instead of nil
	setupProjectCleanup(t, projectManager, createdProject, volumes)

	// Step 2.5: Verify project status after deployment
	t.Log("Step 2.5: Verifying project status after deployment...")

	status, err := projectManager.GetStatus(createdProject.ID)
	require.NoError(t, err, "Getting status should succeed")
	require.NotNil(t, status, "Status should not be nil")

	// Project should be running after deployment
	assert.Equal(t, ComposeProjectStatusRunning, status.Status, "Project should be running after deployment")
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

	config, stderr, err := projectManager.GetConfig(createdProject.ID)
	require.NoError(t, err, "Getting config should succeed")
	require.NotEmpty(t, config, "Config should not be empty")
	t.Logf("Config stderr: %s", stderr)

	// Verify config contains expected compose content
	assert.Contains(t, config, "services:", "Config should contain services section")
	t.Logf("Project configuration retrieved (%d characters)", len(config))

	// Step 4: Get project logs using streaming
	t.Log("Step 4: Getting project logs with streaming...")

	logsChan := make(chan StreamMessage, 100)
	logsDone := make(chan error, 1)

	// Get logs (no streaming needed)

	go func() {
		defer close(logsChan)
		// Get static logs instead of streaming
		stdout, stderr, err := projectManager.GetLogs(createdProject.ID)
		if err != nil {
			logsDone <- err
			return
		}
		// Send stdout lines first
		lines := strings.Split(stdout, "\n")
		for _, line := range lines {
			if line != "" {
				logsChan <- StreamMessage{Type: "stdout", Content: line}
			}
		}
		// Send stderr lines if any
		if stderr != "" {
			stderrLines := strings.Split(stderr, "\n")
			for _, line := range stderrLines {
				if line != "" {
					logsChan <- StreamMessage{Type: "stderr", Content: line}
				}
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
			logMessages = append(logMessages, msg.Content)
			t.Logf("Log output [%s]: %s", msg.Type, strings.TrimSpace(msg.Content))

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

	stopChan := make(chan StreamMessage, 100)
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
			stopMessages = append(stopMessages, msg.Content)
			t.Logf("Stop output [%s]: %s", msg.Type, strings.TrimSpace(msg.Content))
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
	assert.Equal(t, ComposeProjectStatusStopped, stoppedStatus.Status, "Project should be stopped after stop operation")
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

	err = projectManager.Remove(createdProject.ID, true)
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
	outputChan := make(chan StreamMessage, 100)
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
			t.Logf("Deploy output: %s", strings.TrimSpace(msg.Content))
		case err := <-deployDone:
			require.NoError(t, err, "Deployment should succeed")
			break deployLoop
		case <-timeout:
			t.Fatal("Deployment timed out")
		}
	}

	// IMMEDIATELY capture volumes after deployment (regardless of outcome)
	volumes := getProjectNamedVolumes(t, createdProject)
	if len(volumes) > 0 {
		t.Logf("Captured %d named volumes after deployment: %v", len(volumes), volumes)
	}

	// Update cleanup to use captured volumes instead of nil
	setupProjectCleanup(t, projectManager, createdProject, volumes)

	// Verify project status after deployment (merge strategy should have web, redis, db)
	t.Log("Verifying merge strategy project status...")

	status, err := projectManager.GetStatus(createdProject.ID)
	require.NoError(t, err, "Getting status should succeed")
	require.NotNil(t, status, "Status should not be nil")

	// Project should be running
	assert.Equal(t, ComposeProjectStatusRunning, status.Status, "Project should be running after deployment")

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
	config, stderr, err := projectManager.GetConfig(createdProject.ID)
	require.NoError(t, err, "Getting config should succeed")
	require.NotEmpty(t, config, "Config should not be empty")
	t.Logf("Config stderr: %s", stderr)

	// Config should show merged result from both files
	assert.Contains(t, config, "services:", "Config should contain services section")
	t.Logf("Merge strategy configuration verified (%d characters)", len(config))

	// Cleanup
	err = projectManager.StopStreaming(createdProject.ID, make(chan StreamMessage, 100))
	require.NoError(t, err, "Stopping should succeed")

	err = projectManager.Remove(createdProject.ID, true)
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
	outputChan := make(chan StreamMessage, 100)
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
			t.Logf("Deploy output: %s", strings.TrimSpace(msg.Content))
		case err := <-deployDone:
			require.NoError(t, err, "Deployment should succeed")
			break deployLoop
		case <-timeout:
			t.Fatal("Deployment timed out")
		}
	}

	// IMMEDIATELY capture volumes after deployment (regardless of outcome)
	volumes := getProjectNamedVolumes(t, createdProject)
	if len(volumes) > 0 {
		t.Logf("Captured %d named volumes after deployment: %v", len(volumes), volumes)
	}

	// Update cleanup to use captured volumes instead of nil
	setupProjectCleanup(t, projectManager, createdProject, volumes)

	// Verify extended configuration
	config, stderr, err := projectManager.GetConfig(createdProject.ID)
	require.NoError(t, err, "Getting config should succeed")
	require.NotEmpty(t, config, "Config should not be empty")
	t.Logf("Config stderr: %s", stderr)

	assert.Contains(t, config, "services:", "Config should contain services section")
	t.Logf("Extend strategy configuration verified (%d characters)", len(config))

	// Cleanup
	err = projectManager.StopStreaming(createdProject.ID, make(chan StreamMessage, 100))
	require.NoError(t, err, "Stopping should succeed")

	err = projectManager.Remove(createdProject.ID, true)
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
	outputChan := make(chan StreamMessage, 100)
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
			t.Logf("Deploy output: %s", strings.TrimSpace(msg.Content))
		case err := <-deployDone:
			require.NoError(t, err, "Deployment should succeed")
			break deployLoop
		case <-timeout:
			t.Fatal("Deployment timed out")
		}
	}

	// IMMEDIATELY capture volumes after deployment (regardless of outcome)
	volumes := getProjectNamedVolumes(t, createdProject)
	if len(volumes) > 0 {
		t.Logf("Captured %d named volumes after deployment: %v", len(volumes), volumes)
	}

	// Update cleanup to use captured volumes instead of nil
	setupProjectCleanup(t, projectManager, createdProject, volumes)

	// Verify included configuration
	config, stderr, err := projectManager.GetConfig(createdProject.ID)
	require.NoError(t, err, "Getting config should succeed")
	require.NotEmpty(t, config, "Config should not be empty")
	t.Logf("Config stderr: %s", stderr)

	assert.Contains(t, config, "services:", "Config should contain services section")
	t.Logf("Include strategy configuration verified (%d characters)", len(config))

	// Cleanup
	err = projectManager.StopStreaming(createdProject.ID, make(chan StreamMessage, 100))
	require.NoError(t, err, "Stopping should succeed")

	err = projectManager.Remove(createdProject.ID, true)
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
	outputChan := make(chan StreamMessage, 100)
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
			t.Logf("Deploy output: %s", strings.TrimSpace(msg.Content))
		case err := <-deployDone:
			require.NoError(t, err, "Deployment should succeed")
			break deployLoop
		case <-timeout:
			t.Fatal("Deployment timed out")
		}
	}

	// IMMEDIATELY capture volumes after deployment (regardless of outcome)
	volumes := getProjectNamedVolumes(t, createdProject)
	if len(volumes) > 0 {
		t.Logf("Captured %d named volumes after deployment: %v", len(volumes), volumes)
	}

	// Update cleanup to use captured volumes instead of nil
	setupProjectCleanup(t, projectManager, createdProject, volumes)

	// Verify variable interpolation worked
	config, stderr, err := projectManager.GetConfig(createdProject.ID)
	require.NoError(t, err, "Getting config should succeed")
	require.NotEmpty(t, config, "Config should not be empty")
	t.Logf("Config stderr: %s", stderr)

	// Config should show interpolated values
	assert.Contains(t, config, "services:", "Config should contain services section")
	assert.Contains(t, config, "3000", "Config should contain interpolated port value")
	assert.Contains(t, config, "latest", "Config should contain interpolated version value")
	assert.Contains(t, config, "test-value-123", "Config should contain interpolated custom value")
	t.Logf("Variable interpolation verified in configuration (%d characters)", len(config))

	// Cleanup
	err = projectManager.StopStreaming(createdProject.ID, make(chan StreamMessage, 100))
	require.NoError(t, err, "Stopping should succeed")

	err = projectManager.Remove(createdProject.ID, true)
	require.NoError(t, err, "Project removal should succeed")

	t.Logf("Variables test completed successfully")
}

// TestProjectManager_Integration_VolumeMounts tests the volume permission initialization feature
// using a comprehensive test suite with 23 different services covering various volume mount scenarios
func TestProjectManager_Integration_VolumeMounts(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	service, _ := setupProjectService(t)
	projectManager := ProjectManager(service)

	testRepoURL := "https://github.com/oar-cd/test-project.git"
	testProjectName := "test-project-volume-mounts"

	t.Log("Testing volume mounts with permission initialization...")

	project := &Project{
		ID:        uuid.New(),
		Name:      testProjectName,
		GitURL:    testRepoURL,
		GitBranch: "volume-mounts", // Use volume-mounts branch with comprehensive test suite
		GitAuth:   nil,
		// Using compose file with 23 different volume mount test cases
		ComposeFiles: []string{"compose.yaml"},
	}

	createdProject, err := projectManager.Create(project)
	require.NoError(t, err, "Project creation should succeed")
	require.NotNil(t, createdProject, "Created project should not be nil")

	t.Logf("Project created successfully with ID: %s", createdProject.ID)

	// Deploy the project
	t.Log("Deploying project...")

	outputChan := make(chan StreamMessage, 200)
	deployDone := make(chan error, 1)

	go func() {
		defer close(outputChan)
		deployDone <- projectManager.DeployStreaming(createdProject.ID, false, outputChan)
	}()

	// Wait for deployment to complete and consume output
	timeout := time.After(180 * time.Second)

deployLoop:
	for {
		select {
		case msg, ok := <-outputChan:
			if !ok {
				break deployLoop
			}
			t.Logf("Deploy: %s", strings.TrimSpace(msg.Content))
		case err := <-deployDone:
			require.NoError(t, err, "Deployment should succeed")
			break deployLoop
		case <-timeout:
			t.Fatal("Deployment timed out")
		}
	}

	// IMMEDIATELY capture volumes after deployment (regardless of outcome)
	volumes := getProjectNamedVolumes(t, createdProject)
	if len(volumes) > 0 {
		t.Logf("Captured %d named volumes after deployment: %v", len(volumes), volumes)
	}

	// Update cleanup to use captured volumes instead of nil
	setupProjectCleanup(t, projectManager, createdProject, volumes)

	// Wait for containers to stabilize and verify all are running
	t.Log("Waiting for all containers to be running...")

	var status *ComposeStatus
	maxWaitTime := 60 * time.Second
	checkInterval := 5 * time.Second
	waitTimeout := time.After(maxWaitTime)
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	expectedServiceCount := 23

	for {
		select {
		case <-waitTimeout:
			t.Fatal("Timed out waiting for all containers to be running")
		case <-ticker.C:
			status, err = projectManager.GetStatus(createdProject.ID)
			require.NoError(t, err, "Getting status should succeed")
			require.NotNil(t, status, "Status should not be nil")

			if status.Status == ComposeProjectStatusRunning {
				// Check if all containers are running
				allRunning := true
				runningCount := 0
				for _, container := range status.Containers {
					if container.State == "running" {
						runningCount++
					} else {
						allRunning = false
					}
				}

				if allRunning && len(status.Containers) == expectedServiceCount {
					t.Logf("All %d containers are running", runningCount)
					goto containersReady
				} else {
					t.Logf("Waiting... %d/%d containers running", runningCount, len(status.Containers))
				}
			} else {
				t.Logf("Project status: %s", status.Status)
			}
		}
	}

containersReady:

	// Verify expected test services are present
	serviceNames := make([]string, len(status.Containers))
	for i, container := range status.Containers {
		serviceNames[i] = container.Service
	}

	expectedServices := []string{
		"no-user-with-volumes",
		"root-user-with-volumes",
		"nonroot-user-with-volumes",
		"image-defaults-nonroot",
		"opensearch-nonroot",
		"no-volumes",
		"build-no-user",
		"build-root-user",
		"build-nonroot-user",
		"both-build-image-no-user",
		"both-build-image-root",
		"both-build-image-nonroot",
		"both-build-image-defaults",
		"named-volumes-only",
		"bind-mounts-only",
		"mixed-volumes",
		"file-bind-mount-existing",
		"file-bind-mount-nonexisting",
		"directory-bind-mount",
		"multiple-volumes",
		"user-group-format",
		"numeric-user",
		"complex-paths",
	}

	for _, expectedService := range expectedServices {
		assert.Contains(t, serviceNames, expectedService, "Should have service: %s", expectedService)
	}

	t.Logf("All expected volume mount test services verified: %d services", len(expectedServices))

	// Cleanup: Stop and remove containers
	t.Log("Stopping and removing containers...")

	stopChan := make(chan StreamMessage, 100)
	stopDone := make(chan error, 1)

	go func() {
		defer close(stopChan)
		stopDone <- projectManager.StopStreaming(createdProject.ID, stopChan)
	}()

	// Wait for stop to complete and show output
	stopTimeout := time.After(60 * time.Second)

stopLoop:
	for {
		select {
		case msg, ok := <-stopChan:
			if !ok {
				break stopLoop
			}
			t.Logf("Stop: %s", strings.TrimSpace(msg.Content))
		case err := <-stopDone:
			require.NoError(t, err, "Stopping should succeed")
			break stopLoop
		case <-stopTimeout:
			t.Fatal("Stop operation timed out")
		}
	}

	// Remove the project (this should clean up everything)
	err = projectManager.Remove(createdProject.ID, true)
	require.NoError(t, err, "Project removal should succeed")

	t.Logf("Volume mounts integration test completed successfully")
}

// setupProjectCleanup registers a cleanup function that ensures the project is properly removed
// regardless of test outcome, including Docker resources and bind mount files
func setupProjectCleanup(
	t *testing.T,
	projectManager ProjectManager,
	createdProject *Project,
	capturedVolumes []string,
) {
	t.Cleanup(func() {
		t.Logf("Cleaning up test project: %s", createdProject.ID)

		// Try to get the project from database first
		_, err := projectManager.Get(createdProject.ID)
		if err == nil {
			// Project still exists in database - normal cleanup
			t.Logf("Project %s found in database, removing normally", createdProject.ID)

			if removeErr := projectManager.Remove(createdProject.ID, true); removeErr != nil {
				t.Logf("Warning: Failed to remove existing project during cleanup: %v", removeErr)
			}

			// Always cleanup bind mount files after removal
			cleanupBindMountFiles(t, createdProject)
		} else {
			// Project not in database - fallback cleanup logic
			t.Logf("Project %s not found in database, checking for deleted directory", createdProject.ID)

			deletedDirPath := GetDeletedDirectoryPath(createdProject.WorkingDir)

			if _, statErr := os.Stat(deletedDirPath); statErr == nil {
				t.Logf("Found deleted directory at %s, creating temporary project for cleanup", deletedDirPath)

				// Create temporary project with SAME ID pointing to deleted directory
				tempProject := &Project{
					ID:           createdProject.ID, // Use original ID, not uuid.New()
					Name:         fmt.Sprintf("cleanup-%s", createdProject.Name),
					GitURL:       createdProject.GitURL,
					GitBranch:    createdProject.GitBranch,
					WorkingDir:   deletedDirPath,
					ComposeFiles: createdProject.ComposeFiles,
					Variables:    createdProject.Variables,
					Status:       ProjectStatusStopped,
				}

				// Get the project repository from the project manager (cast to ProjectService)
				if projectService, ok := projectManager.(*ProjectService); ok {
					// Create project directly in database (skip Git operations)
					_, createErr := projectService.projectRepository.Create(tempProject)
					if createErr != nil {
						t.Logf("Warning: Failed to create temporary project in database for cleanup: %v", createErr)
						return
					}

					t.Logf("Created temporary project %s in database for cleanup, calling Remove", tempProject.ID)

					if removeErr := projectManager.Remove(tempProject.ID, true); removeErr != nil {
						t.Logf("Warning: Failed to remove temporary project during cleanup: %v", removeErr)
					}
				} else {
					t.Logf("Warning: Could not cast projectManager to *ProjectService for cleanup")
				}

				cleanupBindMountFiles(t, tempProject)
			} else {
				t.Logf("No deleted directory found at %s, project may have been fully cleaned up already", deletedDirPath)
			}
		}

		// Finally, explicitly clean up any captured volumes
		if len(capturedVolumes) > 0 {
			t.Logf("Explicitly removing %d captured volumes", len(capturedVolumes))
			for _, volume := range capturedVolumes {
				if err := removeDockerVolume(t, volume); err != nil {
					t.Logf("Warning: Failed to remove volume %s: %v", volume, err)
				} else {
					t.Logf("Successfully removed volume: %s", volume)
				}
			}
		}
	})
}

// cleanupBindMountFiles uses a privileged container to clean up bind mount files
// that may be owned by different users, preventing permission denied errors during test cleanup
func cleanupBindMountFiles(t *testing.T, project *Project) {
	// After project removal, the working directory is renamed with "deleted-" prefix
	// So we need to construct the path to the git directory within the deleted project
	workingDir := GetDeletedDirectoryPath(project.WorkingDir)
	gitDir := filepath.Join(workingDir, "git")

	// Use a helper container with root privileges to clean up bind mount files
	// This ensures we can delete files created by any user (including root)
	containerName := fmt.Sprintf("cleanup-%s", project.Name)

	t.Logf("Cleaning up bind mount files in %s using helper container", gitDir)

	// Create a simple Docker client for cleanup
	dockerClient, err := NewDockerClient()
	if err != nil {
		t.Logf("Warning: Could not create Docker client for cleanup: %v", err)
		return
	}
	defer func() {
		if closeErr := dockerClient.Close(); closeErr != nil {
			t.Logf("Warning: Failed to close Docker client: %v", closeErr)
		}
	}()

	// Get current user's UID and GID for ownership change
	currentUID := os.Getuid()
	currentGID := os.Getgid()

	// Mount the git directory and recursively change ownership back to current user
	// Also ensure files/directories have proper permissions for cleanup
	cleanupCommand := fmt.Sprintf("chown -R %d:%d /cleanup && chmod -R u+rw /cleanup",
		currentUID, currentGID)

	containerMounts := []mount.Mount{
		{
			Type:   mount.TypeBind,
			Source: gitDir,
			Target: "/cleanup",
		},
	}

	// Run cleanup container as root to ensure we can delete any files
	err = dockerClient.RunVolumeChowningContainer(containerName, cleanupCommand, containerMounts)
	if err != nil {
		t.Logf("Warning: Bind mount cleanup failed (this may cause temp directory cleanup issues): %v", err)
	} else {
		t.Logf("Successfully cleaned up bind mount files")
	}
}

// getProjectNamedVolumes inspects running containers for the project and returns all named volume names
// using Docker SDK instead of shelling out to docker-compose
func getProjectNamedVolumes(t *testing.T, project *Project) []string {
	dockerClient, err := NewDockerClient()
	if err != nil {
		t.Logf("Failed to create Docker client for volume inspection: %v", err)
		return nil
	}
	defer func() {
		if closeErr := dockerClient.Close(); closeErr != nil {
			t.Logf("Warning: Failed to close Docker client: %v", closeErr)
		}
	}()

	ctx := context.Background()

	// Filter containers by compose project label
	options := container.ListOptions{
		Filters: filters.NewArgs(
			filters.KeyValuePair{
				Key:   "label",
				Value: fmt.Sprintf("com.docker.compose.project=%s", project.Name),
			},
		),
	}

	containers, err := dockerClient.cli.ContainerList(ctx, options)
	if err != nil {
		t.Logf("Failed to list containers for project %s: %v", project.Name, err)
		return nil
	}

	if len(containers) == 0 {
		t.Logf("No containers found for project %s", project.Name)
		return nil
	}

	var volumes []string
	for _, container := range containers {
		// Inspect each container to get its volume mounts
		containerJSON, err := dockerClient.cli.ContainerInspect(ctx, container.ID)
		if err != nil {
			t.Logf("Failed to inspect container %s: %v", container.ID, err)
			continue
		}

		// Extract volume names from mounts (skip bind mounts)
		for _, mount := range containerJSON.Mounts {
			if mount.Type == "volume" && mount.Name != "" {
				// Collect all named volumes (both anonymous and explicit)
				volumes = append(volumes, mount.Name)
			}
		}
	}

	// Remove duplicates
	volumeSet := make(map[string]bool)
	var uniqueVolumes []string
	for _, volume := range volumes {
		if !volumeSet[volume] {
			volumeSet[volume] = true
			uniqueVolumes = append(uniqueVolumes, volume)
		}
	}

	return uniqueVolumes
}

// removeDockerVolume removes a Docker volume using Docker SDK with force option
func removeDockerVolume(t *testing.T, volumeName string) error {
	dockerClient, err := NewDockerClient()
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %v", err)
	}
	defer func() {
		if closeErr := dockerClient.Close(); closeErr != nil {
			t.Logf("Warning: Failed to close Docker client: %v", closeErr)
		}
	}()

	ctx := context.Background()
	// Use force=true to not fail on non-existent volumes
	err = dockerClient.cli.VolumeRemove(ctx, volumeName, true)
	if err != nil {
		return fmt.Errorf("failed to remove volume %s: %v", volumeName, err)
	}
	return nil
}
