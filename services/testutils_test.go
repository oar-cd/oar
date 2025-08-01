package services

import (
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fernet/fernet-go"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/ch00k/oar/internal/dbutil"
	"github.com/ch00k/oar/models"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *gorm.DB {
	// Initialize database using shared utility
	database, err := dbutil.InitDatabase(dbutil.DBConfig{
		Path:     ":memory:",
		LogLevel: logger.Silent,
	})
	if err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}

	// Run migrations for all models (single source of truth)
	err = models.AutoMigrateAll(database)
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
func setupTestEncryption(t *testing.T) *EncryptionService {
	encryption, err := NewEncryptionService(generateTestKey())
	if err != nil {
		t.Fatalf("Failed to create test encryption service: %v", err)
	}
	return encryption
}

// createTestProject creates a standard test project for domain layer testing
func createTestProject() *Project {
	return &Project{
		ID:               uuid.New(),
		Name:             "test-project",
		GitURL:           "https://github.com/test/repo.git",
		WorkingDir:       "/tmp/test-project",
		ComposeFiles:     []string{"docker-compose.yml"},
		EnvironmentFiles: []string{".env"},
		Status:           ProjectStatusStopped,
		LastCommit:       stringPtr("abc123"),
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
}

// createTestProjectWithOptions creates a test project with customizable options
func createTestProjectWithOptions(opts ProjectOptions) *Project {
	project := createTestProject()

	if opts.Name != "" {
		project.Name = opts.Name
	}
	if opts.GitURL != "" {
		project.GitURL = opts.GitURL
	}
	if opts.WorkingDir != "" {
		project.WorkingDir = opts.WorkingDir
	}
	if len(opts.ComposeFiles) > 0 {
		project.ComposeFiles = opts.ComposeFiles
	}
	if len(opts.EnvironmentFiles) > 0 {
		project.EnvironmentFiles = opts.EnvironmentFiles
	}
	if opts.Status != ProjectStatusUnknown {
		project.Status = opts.Status
	}

	return project
}

// ProjectOptions provides customization options for test projects
type ProjectOptions struct {
	Name             string
	GitURL           string
	WorkingDir       string
	ComposeFiles     []string
	EnvironmentFiles []string
	Status           ProjectStatus
}

// createTestDeployment creates a test deployment for domain layer testing
func createTestDeployment(projectID uuid.UUID) *Deployment {
	return &Deployment{
		ID:         uuid.New(),
		ProjectID:  projectID,
		CommitHash: "abc123",
		Status:     DeploymentStatusCompleted,
		Output:     "Deployment successful",
	}
}

// createTestComposeProject creates a compose project for compose service testing
func createTestComposeProject() *ComposeProject {
	// Create a test project
	testProject := &Project{
		ID:               uuid.New(),
		Name:             "test-project",
		GitURL:           "https://github.com/test/repo.git",
		WorkingDir:       "/tmp/test-compose-project",
		ComposeFiles:     []string{"docker-compose.yml"},
		EnvironmentFiles: []string{".env"},
		Status:           ProjectStatusStopped,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	// Create test config
	config := &Config{
		DataDir:       "/tmp",
		LogLevel:      "info",
		ColorEnabled:  false,
		DockerCommand: "docker",
		DockerHost:    "unix:///var/run/docker.sock",
		GitTimeout:    5 * time.Minute,
	}

	return NewComposeProject(testProject, config)
}

// setupProjectService creates a project service with mock dependencies for testing
func setupProjectService(
	t *testing.T,
) (*ProjectService, *MockProjectRepository, *MockDeploymentRepository, *MockGitExecutor, string) {
	tempDir := t.TempDir()

	projectRepo := &MockProjectRepository{
		projects: make(map[uuid.UUID]*Project),
	}
	deploymentRepo := &MockDeploymentRepository{
		deployments: make(map[uuid.UUID]*Deployment),
	}
	gitService := &MockGitExecutor{}

	config := &Config{
		DataDir:      tempDir,
		WorkspaceDir: filepath.Join(tempDir, "projects"),
	}

	// Ensure workspace directory exists
	err := os.MkdirAll(config.WorkspaceDir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create workspace dir: %v", err)
	}

	service := &ProjectService{
		projectRepository:    projectRepo,
		deploymentRepository: deploymentRepo,
		gitService:           gitService,
		config:               config,
	}

	return service, projectRepo, deploymentRepo, gitService, tempDir
}

// Utility functions
func stringPtr(s string) *string {
	return &s
}

// Mock repositories for testing
type MockProjectRepository struct {
	projects map[uuid.UUID]*Project
}

func (m *MockProjectRepository) List() ([]*Project, error) {
	projects := make([]*Project, 0, len(m.projects))
	for _, project := range m.projects {
		projects = append(projects, project)
	}
	return projects, nil
}

func (m *MockProjectRepository) FindByID(id uuid.UUID) (*Project, error) {
	if project, exists := m.projects[id]; exists {
		return project, nil
	}
	return nil, errors.New("project not found")
}

func (m *MockProjectRepository) FindByName(name string) (*Project, error) {
	for _, project := range m.projects {
		if project.Name == name {
			return project, nil
		}
	}
	return nil, errors.New("project not found")
}

func (m *MockProjectRepository) Create(project *Project) (*Project, error) {
	if project.ID == uuid.Nil {
		project.ID = uuid.New()
	}

	// Check for duplicate names
	for _, existing := range m.projects {
		if existing.Name == project.Name {
			return nil, errors.New("project already exists")
		}
	}

	now := time.Now()
	project.CreatedAt = now
	project.UpdatedAt = now

	m.projects[project.ID] = project
	return project, nil
}

func (m *MockProjectRepository) Update(project *Project) error {
	if _, exists := m.projects[project.ID]; !exists {
		return errors.New("project not found")
	}

	project.UpdatedAt = time.Now()
	m.projects[project.ID] = project
	return nil
}

func (m *MockProjectRepository) Delete(id uuid.UUID) error {
	if _, exists := m.projects[id]; !exists {
		return errors.New("project not found")
	}
	delete(m.projects, id)
	return nil
}

type MockDeploymentRepository struct {
	deployments map[uuid.UUID]*Deployment
}

func (m *MockDeploymentRepository) FindByID(id uuid.UUID) (*Deployment, error) {
	if deployment, exists := m.deployments[id]; exists {
		return deployment, nil
	}
	return nil, errors.New("deployment not found")
}

func (m *MockDeploymentRepository) Create(deployment *Deployment) error {
	if deployment.ID == uuid.Nil {
		deployment.ID = uuid.New()
	}
	m.deployments[deployment.ID] = deployment
	return nil
}

func (m *MockDeploymentRepository) ListByProjectID(projectID uuid.UUID) ([]*Deployment, error) {
	var deployments []*Deployment
	for _, deployment := range m.deployments {
		if deployment.ProjectID == projectID {
			deployments = append(deployments, deployment)
		}
	}
	return deployments, nil
}
