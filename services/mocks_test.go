package services

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/oar-cd/oar/models"
	"github.com/stretchr/testify/mock"
)

// MockGitExecutor for testing
type MockGitExecutor struct {
	CloneFunc                 func(gitURL string, gitBranch string, gitAuth *GitAuthConfig, workingDir string) error
	PullFunc                  func(gitBranch string, auth *GitAuthConfig, workingDir string) error
	FetchFunc                 func(gitBranch string, gitAuth *GitAuthConfig, workingDir string) error
	GetLatestCommitFunc       func(workingDir string) (string, error)
	GetRemoteLatestCommitFunc func(workingDir string, gitBranch string) (string, error)
	TestAuthenticationFunc    func(gitURL string, gitAuth *GitAuthConfig) error
	GetDefaultBranchFunc      func(gitURL string, gitAuth *GitAuthConfig) (string, error)
}

func (m *MockGitExecutor) Clone(gitURL string, gitBranch string, gitAuth *GitAuthConfig, workingDir string) error {
	if m.CloneFunc != nil {
		return m.CloneFunc(gitURL, gitBranch, gitAuth, workingDir)
	}
	return nil
}

func (m *MockGitExecutor) Pull(gitBranch string, gitAuth *GitAuthConfig, workingDir string) error {
	if m.PullFunc != nil {
		return m.PullFunc(gitBranch, gitAuth, workingDir)
	}
	return nil
}

func (m *MockGitExecutor) Fetch(gitBranch string, gitAuth *GitAuthConfig, workingDir string) error {
	if m.FetchFunc != nil {
		return m.FetchFunc(gitBranch, gitAuth, workingDir)
	}
	return nil
}

func (m *MockGitExecutor) GetLatestCommit(workingDir string) (string, error) {
	if m.GetLatestCommitFunc != nil {
		return m.GetLatestCommitFunc(workingDir)
	}
	return "mock-commit-hash", nil
}

func (m *MockGitExecutor) GetRemoteLatestCommit(workingDir string, gitBranch string) (string, error) {
	if m.GetRemoteLatestCommitFunc != nil {
		return m.GetRemoteLatestCommitFunc(workingDir, gitBranch)
	}
	return "mock-remote-commit-hash", nil
}

func (m *MockGitExecutor) TestAuthentication(gitURL string, gitAuth *GitAuthConfig) error {
	if m.TestAuthenticationFunc != nil {
		return m.TestAuthenticationFunc(gitURL, gitAuth)
	}
	return nil
}

func (m *MockGitExecutor) GetDefaultBranch(gitURL string, gitAuth *GitAuthConfig) (string, error) {
	if m.GetDefaultBranchFunc != nil {
		return m.GetDefaultBranchFunc(gitURL, gitAuth)
	}
	return "main", nil // Default mock return value
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

// MockComposeProject implements ComposeProjectInterface for testing
type MockComposeProject struct {
	mock.Mock
}

func (m *MockComposeProject) Up() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockComposeProject) Down() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockComposeProject) Logs() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockComposeProject) GetConfig() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockComposeProject) Status() (*ComposeStatus, error) {
	args := m.Called()
	return args.Get(0).(*ComposeStatus), args.Error(1)
}

func (m *MockComposeProject) UpStreaming(outputChan chan<- string) error {
	args := m.Called(outputChan)
	return args.Error(0)
}

func (m *MockComposeProject) UpPiping() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockComposeProject) DownStreaming(outputChan chan<- string) error {
	args := m.Called(outputChan)
	return args.Error(0)
}

func (m *MockComposeProject) DownPiping() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockComposeProject) LogsStreaming(outputChan chan<- string) error {
	args := m.Called(outputChan)
	return args.Error(0)
}

func (m *MockComposeProject) LogsPiping() error {
	args := m.Called()
	return args.Error(0)
}

// MockProjectManager implements the ProjectManager interface for testing
type MockProjectManager struct {
	ListFunc             func() ([]*Project, error)
	GetFunc              func(id uuid.UUID) (*Project, error)
	CreateFunc           func(project *Project) (*Project, error)
	UpdateFunc           func(project *Project) error
	RemoveFunc           func(projectID uuid.UUID) error
	DeployStreamingFunc  func(projectID uuid.UUID, pull bool, outputChan chan<- string) error
	DeployPipingFunc     func(projectID uuid.UUID, pull bool) error
	StopFunc             func(projectID uuid.UUID) error
	StopStreamingFunc    func(projectID uuid.UUID, outputChan chan<- string) error
	StopPipingFunc       func(projectID uuid.UUID) error
	GetLogsStreamingFunc func(projectID uuid.UUID, outputChan chan<- string) error
	GetLogsPipingFunc    func(projectID uuid.UUID) error
	GetConfigFunc        func(projectID uuid.UUID) (string, error)
	GetStatusFunc        func(projectID uuid.UUID) (*ComposeStatus, error)
	ListDeploymentsFunc  func(projectID uuid.UUID) ([]*Deployment, error)
}

func (m *MockProjectManager) List() ([]*Project, error) {
	if m.ListFunc != nil {
		return m.ListFunc()
	}
	return []*Project{}, nil
}

func (m *MockProjectManager) Get(id uuid.UUID) (*Project, error) {
	if m.GetFunc != nil {
		return m.GetFunc(id)
	}
	return &Project{ID: id}, nil
}

func (m *MockProjectManager) Create(project *Project) (*Project, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(project)
	}
	return project, nil
}

func (m *MockProjectManager) Update(project *Project) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(project)
	}
	return nil
}

func (m *MockProjectManager) Remove(projectID uuid.UUID) error {
	if m.RemoveFunc != nil {
		return m.RemoveFunc(projectID)
	}
	return nil
}

func (m *MockProjectManager) DeployStreaming(projectID uuid.UUID, pull bool, outputChan chan<- string) error {
	if m.DeployStreamingFunc != nil {
		return m.DeployStreamingFunc(projectID, pull, outputChan)
	}
	return nil
}

func (m *MockProjectManager) DeployPiping(projectID uuid.UUID, pull bool) error {
	if m.DeployPipingFunc != nil {
		return m.DeployPipingFunc(projectID, pull)
	}
	return nil
}

func (m *MockProjectManager) Stop(projectID uuid.UUID) error {
	if m.StopFunc != nil {
		return m.StopFunc(projectID)
	}
	return nil
}

func (m *MockProjectManager) StopStreaming(projectID uuid.UUID, outputChan chan<- string) error {
	if m.StopStreamingFunc != nil {
		return m.StopStreamingFunc(projectID, outputChan)
	}
	return nil
}

func (m *MockProjectManager) StopPiping(projectID uuid.UUID) error {
	if m.StopPipingFunc != nil {
		return m.StopPipingFunc(projectID)
	}
	return nil
}

func (m *MockProjectManager) GetLogsStreaming(projectID uuid.UUID, outputChan chan<- string) error {
	if m.GetLogsStreamingFunc != nil {
		return m.GetLogsStreamingFunc(projectID, outputChan)
	}
	return nil
}

func (m *MockProjectManager) GetLogsPiping(projectID uuid.UUID) error {
	if m.GetLogsPipingFunc != nil {
		return m.GetLogsPipingFunc(projectID)
	}
	return nil
}

func (m *MockProjectManager) GetConfig(projectID uuid.UUID) (string, error) {
	if m.GetConfigFunc != nil {
		return m.GetConfigFunc(projectID)
	}
	return "mock config", nil
}

func (m *MockProjectManager) GetStatus(projectID uuid.UUID) (*ComposeStatus, error) {
	if m.GetStatusFunc != nil {
		return m.GetStatusFunc(projectID)
	}
	return &ComposeStatus{}, nil
}

func (m *MockProjectManager) ListDeployments(projectID uuid.UUID) ([]*Deployment, error) {
	if m.ListDeploymentsFunc != nil {
		return m.ListDeploymentsFunc(projectID)
	}
	return []*Deployment{}, nil
}
