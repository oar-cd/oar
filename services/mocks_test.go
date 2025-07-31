package services

import (
	"fmt"
	"time"

	"github.com/ch00k/oar/models"
	"github.com/google/uuid"
)

// MockGitExecutor for testing
type MockGitExecutor struct {
	CloneFunc           func(gitURL, workingDir string, auth *AuthConfig) error
	PullFunc            func(workingDir string, auth *AuthConfig) error
	GetLatestCommitFunc func(workingDir string) (string, error)
}

func (m *MockGitExecutor) Clone(gitURL, workingDir string, auth *AuthConfig) error {
	if m.CloneFunc != nil {
		return m.CloneFunc(gitURL, workingDir, auth)
	}
	return nil
}

func (m *MockGitExecutor) Pull(workingDir string, auth *AuthConfig) error {
	if m.PullFunc != nil {
		return m.PullFunc(workingDir, auth)
	}
	return nil
}

func (m *MockGitExecutor) GetLatestCommit(workingDir string) (string, error) {
	if m.GetLatestCommitFunc != nil {
		return m.GetLatestCommitFunc(workingDir)
	}
	return "mock-commit-hash", nil
}

// MockDockerComposeExecutor for testing
type MockDockerComposeExecutor struct {
	DeployFunc func(name, workingDir, composeFile string, config Deployment) (string, error)
	DownFunc   func(name, workingDir, composeFile string) (string, error)
}

func (m *MockDockerComposeExecutor) Deploy(name, workingDir, composeFile string, config Deployment) (string, error) {
	if m.DeployFunc != nil {
		return m.DeployFunc(name, workingDir, composeFile, config)
	}
	return "mock deploy output", nil
}

func (m *MockDockerComposeExecutor) Down(name, workingDir, composeFile string) (string, error) {
	if m.DownFunc != nil {
		return m.DownFunc(name, workingDir, composeFile)
	}
	return "mock down output", nil
}

// MockDB represents a simple in-memory database for testing
type MockDB struct {
	projects    map[uuid.UUID]*models.ProjectModel
	deployments map[uuid.UUID]*models.DeploymentModel
}

func NewMockDB() *MockDB {
	return &MockDB{
		projects:    make(map[uuid.UUID]*models.ProjectModel),
		deployments: make(map[uuid.UUID]*models.DeploymentModel),
	}
}

func (db *MockDB) CreateProject(project *models.ProjectModel) error {
	if project.ID == uuid.Nil {
		project.ID = uuid.New()
	}
	project.CreatedAt = time.Now()
	project.UpdatedAt = time.Now()

	// Check for name uniqueness
	for _, p := range db.projects {
		if p.Name == project.Name {
			return fmt.Errorf("project name '%s' already exists", project.Name)
		}
	}

	db.projects[project.ID] = project
	return nil
}

func (db *MockDB) FindProject(id uuid.UUID) (*models.ProjectModel, error) {
	project, exists := db.projects[id]
	if !exists {
		return nil, fmt.Errorf("project not found")
	}
	return project, nil
}

func (db *MockDB) ListProjects() ([]*models.ProjectModel, error) {
	projects := make([]*models.ProjectModel, 0, len(db.projects))
	for _, project := range db.projects {
		projects = append(projects, project)
	}
	return projects, nil
}

func (db *MockDB) DeleteProject(id uuid.UUID) error {
	if _, exists := db.projects[id]; !exists {
		return fmt.Errorf("project not found")
	}
	delete(db.projects, id)
	return nil
}

func (db *MockDB) UpdateProject(project *models.ProjectModel) error {
	if _, exists := db.projects[project.ID]; !exists {
		return fmt.Errorf("project not found")
	}
	project.UpdatedAt = time.Now()
	db.projects[project.ID] = project
	return nil
}

func (db *MockDB) CreateDeployment(deployment *models.DeploymentModel) error {
	if deployment.ID == uuid.Nil {
		deployment.ID = uuid.New()
	}
	deployment.CreatedAt = time.Now()
	db.deployments[deployment.ID] = deployment
	return nil
}

func (db *MockDB) UpdateDeployment(deployment *models.DeploymentModel) error {
	if _, exists := db.deployments[deployment.ID]; !exists {
		return fmt.Errorf("deployment not found")
	}
	db.deployments[deployment.ID] = deployment
	return nil
}
