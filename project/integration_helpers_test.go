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
