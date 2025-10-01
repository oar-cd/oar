package project_test

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fernet/fernet-go"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/oar-cd/oar/config"
	"github.com/oar-cd/oar/db"
	"github.com/oar-cd/oar/docker"
	"github.com/oar-cd/oar/domain"
	"github.com/oar-cd/oar/encryption"
	"github.com/oar-cd/oar/git"
	"github.com/oar-cd/oar/logging"
	"github.com/oar-cd/oar/project"
	"github.com/oar-cd/oar/repository"
)

// TestMain sets up global test configuration before running any tests
func TestMain(m *testing.M) {
	// Initialize debug logging for all tests
	logging.InitLogging("debug")

	// Run the tests
	os.Exit(m.Run())
}

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *gorm.DB {
	// Initialize database using shared utility
	database, err := db.InitDatabase(db.DBConfig{
		Path:     ":memory:",
		LogLevel: logger.Silent,
	})
	if err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}

	// Run migrations for all models (single source of truth)
	err = db.AutoMigrateAll(database)
	if err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	return database
}

// generateTestKey generates a new Fernet key for testing
func generateTestKey() string {
	var key fernet.Key
	if _, err := rand.Read(key[:]); err != nil {
		panic(fmt.Sprintf("failed to generate test encryption key: %v", err))
	}
	return key.Encode()
}

// setupTestEncryption creates a test encryption service
func setupTestEncryption(t *testing.T) *encryption.EncryptionService {
	encryptionSvc, err := encryption.NewEncryptionService(generateTestKey())
	if err != nil {
		t.Fatalf("Failed to create test encryption service: %v", err)
	}
	return encryptionSvc
}

// setupProjectService creates a project service with real Git and Docker dependencies for integration testing
func setupProjectService(
	t *testing.T,
) (*project.ProjectService, *git.GitService, repository.ProjectRepository, string) {
	// Create temporary directory for test data
	tempDir := t.TempDir()
	workspaceDir := filepath.Join(tempDir, "projects")

	// Create workspace directory
	err := os.MkdirAll(workspaceDir, 0o755)
	require.NoError(t, err, "Failed to create workspace directory")

	// Setup configuration for integration testing
	cfg := &config.Config{
		DataDir:      tempDir,
		WorkspaceDir: workspaceDir,
		LogLevel:     "debug",          // Show debug logs for volume initialization testing
		GitTimeout:   60 * time.Second, // Allow time for real git operations
	}

	// Setup test database
	database := setupTestDB(t)

	// Setup test encryption service
	encryptionSvc := setupTestEncryption(t)

	// Create repositories (not mocks for integration testing)
	projectRepo := repository.NewProjectRepository(database, encryptionSvc)
	deploymentRepo := repository.NewDeploymentRepository(database)

	// Create Git service (not mock for integration testing)
	gitService := git.NewGitService(cfg)

	// Create project.ProjectService with real dependencies
	projectService := project.NewProjectService(projectRepo, deploymentRepo, gitService, cfg)

	return projectService, gitService, projectRepo, tempDir
}

// testContext holds common test setup
type testContext struct {
	t              *testing.T
	projectService *project.ProjectService
	gitService     *git.GitService
	projectRepo    repository.ProjectRepository
	projectManager project.ProjectManager
	testRepoURL    string
	workspaceDir   string
}

// setupTest creates common test context used by all integration tests
func setupTest(t *testing.T) *testContext {
	projectService, gitService, projectRepo, tempDir := setupProjectService(t)
	return &testContext{
		t:              t,
		projectService: projectService,
		gitService:     gitService,
		projectRepo:    projectRepo,
		projectManager: project.ProjectManager(projectService),
		testRepoURL:    "https://github.com/oar-cd/test-project.git",
		workspaceDir:   filepath.Join(tempDir, "projects"),
	}
}

// deployProject deploys a project with streaming and waits for completion
// Returns error if deployment fails
func (ctx *testContext) deployProject(projectID uuid.UUID, pull bool, timeoutSeconds int) error {
	outputChan := make(chan docker.StreamMessage, 100)
	deployDone := make(chan error, 1)

	go func() {
		defer close(outputChan)
		deployDone <- ctx.projectManager.DeployStreaming(projectID, pull, outputChan)
	}()

	timeout := time.After(time.Duration(timeoutSeconds) * time.Second)

	for {
		select {
		case <-timeout:
			return fmt.Errorf("deployment timed out after %d seconds", timeoutSeconds)
		case msg, ok := <-outputChan:
			if !ok {
				// Channel closed, wait for final error
				return <-deployDone
			}
			ctx.t.Logf("[deploy] %s: %s", msg.Type, msg.Content)
		case err := <-deployDone:
			// Consume any remaining messages
			for msg := range outputChan {
				ctx.t.Logf("[deploy] %s: %s", msg.Type, msg.Content)
			}
			return err
		}
	}
}

// waitForProjectStatus waits for the project to reach the expected status within the timeout
func (ctx *testContext) waitForProjectStatus(
	projectID uuid.UUID,
	expectedStatus docker.ComposeProjectStatus,
	timeout time.Duration,
) error {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	timeoutChan := time.After(timeout)

	for {
		select {
		case <-timeoutChan:
			return fmt.Errorf("timeout waiting for project status to become %s", expectedStatus)
		case <-ticker.C:
			status, err := ctx.projectManager.GetStatus(projectID)
			if err != nil {
				return fmt.Errorf("failed to get project status: %w", err)
			}
			if status.Status == expectedStatus {
				ctx.t.Logf("Project reached status: %s", expectedStatus)
				return nil
			}
			ctx.t.Logf("Waiting for status %s, current status: %s", expectedStatus, status.Status)
		}
	}
}

// setupCleanup registers cleanup for a project including Docker volumes
func (ctx *testContext) setupCleanup(proj *domain.Project) {
	volumes := getProjectNamedVolumes(ctx.t, proj)
	if len(volumes) > 0 {
		ctx.t.Logf("Captured %d named volumes after deployment: %v", len(volumes), volumes)
	}
	setupProjectCleanup(ctx.t, ctx.projectManager, ctx.projectRepo, proj, volumes)
}

// normalizeComposeConfig normalizes a Docker Compose config YAML by replacing the project's
// WorkingDir with a placeholder. This allows comparing configs between test runs that use
// different temporary directories.
//
// The WorkingDir appears in:
// - Bind volume mount sources (e.g., /path/to/workdir/git/data:/app/data)
// - Build contexts (e.g., context: /path/to/workdir/git)
func normalizeComposeConfig(config string, workingDir string) string {
	// The actual git directory path that appears in the config
	gitDir := filepath.Join(workingDir, "git")

	// Replace all occurrences of the git directory with a placeholder
	normalized := strings.ReplaceAll(config, gitDir, "WORKING_DIR")

	return normalized
}

// collectServiceNames extracts service names from container statuses
func collectServiceNames(containers []docker.ContainerInfo) []string {
	serviceNames := make([]string, len(containers))
	for i, container := range containers {
		serviceNames[i] = container.Service
	}
	return serviceNames
}

// verifyContainersRunning verifies that all containers are in running state with all expected fields
func verifyContainersRunning(t *testing.T, containers []docker.ContainerInfo, projectName string) []string {
	serviceNames := make([]string, len(containers))
	for i, container := range containers {
		serviceNames[i] = container.Service
		require.Equal(t, "running", container.State, "Container %s should be in running state", container.Service)
		require.Contains(t, container.Name, projectName, "Container name should contain project name")
		require.NotEmpty(t, container.Status, "Container should have status description")
		require.NotEmpty(t, container.RunningFor, "Container should have running duration")
		require.Equal(t, 0, container.ExitCode, "Container %s should have exit code 0 when running", container.Service)
	}
	return serviceNames
}

// verifyContainersNotRunning verifies that all containers are not in running state
func verifyContainersNotRunning(t *testing.T, containers []docker.ContainerInfo) {
	for _, container := range containers {
		require.NotEqual(
			t,
			"running",
			container.State,
			"Container %s should not be running after stop",
			container.Service,
		)
	}
}

// verifyExpectedServices verifies that all expected services are present in the service names list
func verifyExpectedServices(t *testing.T, serviceNames []string, expectedServices []string) {
	for _, expectedService := range expectedServices {
		require.Contains(t, serviceNames, expectedService, "Should have service: %s", expectedService)
	}
}

// verifyProjectNotInList verifies that the project is not in the list of all projects
func verifyProjectNotInList(t *testing.T, projectManager project.ProjectManager, projectID uuid.UUID) {
	allProjects, err := projectManager.List()
	require.NoError(t, err, "Listing projects should succeed")

	for _, p := range allProjects {
		require.NotEqual(t, projectID, p.ID, "Removed project should not be in list")
	}
}

// consumeStreamingMessages consumes messages from a streaming channel with timeout and logs them
// Returns collected messages and any error from the done channel
func consumeStreamingMessages(
	t *testing.T,
	msgChan <-chan docker.StreamMessage,
	doneChan <-chan error,
	timeout time.Duration,
	logPrefix string,
) ([]string, error) {
	var messages []string
	timeoutChan := time.After(timeout)

	for {
		select {
		case msg, ok := <-msgChan:
			if !ok {
				// Channel closed, return any error from done channel
				return messages, <-doneChan
			}
			messages = append(messages, msg.Content)
			t.Logf("%s [%s]: %s", logPrefix, msg.Type, strings.TrimSpace(msg.Content))
		case err := <-doneChan:
			// Operation completed, consume any remaining messages
			for msg := range msgChan {
				messages = append(messages, msg.Content)
				t.Logf("%s [%s]: %s", logPrefix, msg.Type, strings.TrimSpace(msg.Content))
			}
			return messages, err
		case <-timeoutChan:
			return messages, fmt.Errorf("%s operation timed out", logPrefix)
		}
	}
}

// collectLogMessages collects log messages from a streaming channel up to a limit
// Returns collected messages and any error
func collectLogMessages(
	t *testing.T,
	logsChan <-chan docker.StreamMessage,
	logsDone <-chan error,
	timeout time.Duration,
	maxMessages int,
) ([]string, error) {
	var logMessages []string
	logsTimeout := time.After(timeout)

	for {
		select {
		case msg, ok := <-logsChan:
			if !ok {
				return logMessages, nil
			}
			logMessages = append(logMessages, msg.Content)
			t.Logf("Log output [%s]: %s", msg.Type, strings.TrimSpace(msg.Content))

			if len(logMessages) >= maxMessages {
				return logMessages, nil
			}
		case err := <-logsDone:
			if err != nil {
				t.Logf("Log streaming ended: %v", err)
			}
			return logMessages, err
		case <-logsTimeout:
			t.Log("Log collection timeout reached")
			return logMessages, nil
		}
	}
}

// waitForAllContainersRunning waits for all containers to be running with a ticker
// Returns the final status when all containers are running
func waitForAllContainersRunning(
	t *testing.T,
	projectManager project.ProjectManager,
	projectID uuid.UUID,
	expectedCount int,
	maxWaitTime time.Duration,
) (*docker.ComposeStatus, error) {
	checkInterval := 500 * time.Millisecond
	waitTimeout := time.After(maxWaitTime)
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-waitTimeout:
			return nil, fmt.Errorf("timed out waiting for all containers to be running")
		case <-ticker.C:
			status, err := projectManager.GetStatus(projectID)
			if err != nil {
				return nil, fmt.Errorf("getting status should succeed: %w", err)
			}
			if status == nil {
				return nil, fmt.Errorf("status should not be nil")
			}

			if status.Status == docker.ComposeProjectStatusRunning {
				allRunning := true
				runningCount := 0
				for _, container := range status.Containers {
					if container.State == "running" {
						runningCount++
					} else {
						allRunning = false
					}
				}

				if allRunning && len(status.Containers) == expectedCount {
					t.Logf("All %d containers are running", runningCount)
					return status, nil
				}
				t.Logf("Waiting... %d/%d containers running", runningCount, len(status.Containers))
			} else {
				t.Logf("Project status: %s", status.Status)
			}
		}
	}
}
