package watcher

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/oar-cd/oar/services"
)

// MockProjectManager implements services.ProjectManager interface for testing
type MockProjectManager struct {
	mock.Mock
}

func (m *MockProjectManager) List() ([]*services.Project, error) {
	args := m.Called()
	return args.Get(0).([]*services.Project), args.Error(1)
}

func (m *MockProjectManager) Get(id uuid.UUID) (*services.Project, error) {
	args := m.Called(id)
	return args.Get(0).(*services.Project), args.Error(1)
}

func (m *MockProjectManager) Create(project *services.Project) (*services.Project, error) {
	args := m.Called(project)
	return args.Get(0).(*services.Project), args.Error(1)
}

func (m *MockProjectManager) Update(project *services.Project) error {
	args := m.Called(project)
	return args.Error(0)
}

func (m *MockProjectManager) Remove(projectID uuid.UUID, removeVolumes bool) error {
	args := m.Called(projectID, removeVolumes)
	return args.Error(0)
}

func (m *MockProjectManager) DeployStreaming(
	projectID uuid.UUID,
	pull bool,
	outputChan chan<- services.StreamMessage,
) error {
	args := m.Called(projectID, pull, outputChan)
	return args.Error(0)
}

func (m *MockProjectManager) DeployPiping(projectID uuid.UUID, pull bool) error {
	args := m.Called(projectID, pull)
	return args.Error(0)
}

func (m *MockProjectManager) Stop(projectID uuid.UUID, removeVolumes bool) error {
	args := m.Called(projectID, removeVolumes)
	return args.Error(0)
}

func (m *MockProjectManager) StopStreaming(projectID uuid.UUID, outputChan chan<- services.StreamMessage) error {
	args := m.Called(projectID, outputChan)
	return args.Error(0)
}

func (m *MockProjectManager) StopPiping(projectID uuid.UUID) error {
	args := m.Called(projectID)
	return args.Error(0)
}

func (m *MockProjectManager) GetLogsStreaming(
	ctx context.Context,
	projectID uuid.UUID,
	outputChan chan<- string,
) error {
	args := m.Called(ctx, projectID, outputChan)
	return args.Error(0)
}

func (m *MockProjectManager) GetLogsPiping(projectID uuid.UUID) error {
	args := m.Called(projectID)
	return args.Error(0)
}

func (m *MockProjectManager) GetConfig(projectID uuid.UUID) (string, string, error) {
	args := m.Called(projectID)
	return args.String(0), args.String(1), args.Error(2)
}

func (m *MockProjectManager) GetLogs(projectID uuid.UUID) (string, string, error) {
	args := m.Called(projectID)
	return args.String(0), args.String(1), args.Error(2)
}

func (m *MockProjectManager) GetStatus(projectID uuid.UUID) (*services.ComposeStatus, error) {
	args := m.Called(projectID)
	return args.Get(0).(*services.ComposeStatus), args.Error(1)
}

func (m *MockProjectManager) ListDeployments(projectID uuid.UUID) ([]*services.Deployment, error) {
	args := m.Called(projectID)
	return args.Get(0).([]*services.Deployment), args.Error(1)
}

// MockGitExecutor implements services.GitExecutor interface for testing
type MockGitExecutor struct {
	mock.Mock
}

func (m *MockGitExecutor) Clone(
	gitURL string,
	gitBranch string,
	gitAuth *services.GitAuthConfig,
	workingDir string,
) error {
	args := m.Called(gitURL, gitBranch, gitAuth, workingDir)
	return args.Error(0)
}

func (m *MockGitExecutor) Pull(gitBranch string, gitAuth *services.GitAuthConfig, workingDir string) error {
	args := m.Called(gitBranch, gitAuth, workingDir)
	return args.Error(0)
}

func (m *MockGitExecutor) Fetch(gitBranch string, gitAuth *services.GitAuthConfig, workingDir string) error {
	args := m.Called(gitBranch, gitAuth, workingDir)
	return args.Error(0)
}

func (m *MockGitExecutor) GetLatestCommit(workingDir string) (string, error) {
	args := m.Called(workingDir)
	return args.String(0), args.Error(1)
}

func (m *MockGitExecutor) GetRemoteLatestCommit(workingDir string, gitBranch string) (string, error) {
	args := m.Called(workingDir, gitBranch)
	return args.String(0), args.Error(1)
}

func (m *MockGitExecutor) TestAuthentication(gitURL string, gitAuth *services.GitAuthConfig) error {
	args := m.Called(gitURL, gitAuth)
	return args.Error(0)
}

func (m *MockGitExecutor) GetDefaultBranch(gitURL string, gitAuth *services.GitAuthConfig) (string, error) {
	args := m.Called(gitURL, gitAuth)
	return args.String(0), args.Error(1)
}

// Helper function to create a test project
func createTestProject(
	id uuid.UUID,
	name string,
	status services.ProjectStatus,
	watcherEnabled bool,
	lastCommit string,
) *services.Project {
	workingDir := "/tmp/test-project-" + name
	return &services.Project{
		ID:             id,
		Name:           name,
		GitURL:         "https://github.com/test/" + name + ".git",
		GitBranch:      "main",
		WorkingDir:     workingDir,
		ComposeFiles:   []string{"docker-compose.yml"},
		Variables:      []string{},
		Status:         status,
		WatcherEnabled: watcherEnabled,
		LastCommit:     &lastCommit,
		GitAuth:        nil,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

func TestNewWatcherService(t *testing.T) {
	mockProjectService := &MockProjectManager{}
	mockGitService := &MockGitExecutor{}
	pollInterval := 5 * time.Minute

	service := NewWatcherService(mockProjectService, mockGitService, pollInterval)

	assert.NotNil(t, service)
	assert.Equal(t, mockProjectService, service.projectService)
	assert.Equal(t, mockGitService, service.gitService)
	assert.Equal(t, pollInterval, service.pollInterval)
}

func TestWatcherService_Start_ContextCancellation(t *testing.T) {
	mockProjectService := &MockProjectManager{}
	mockGitService := &MockGitExecutor{}
	pollInterval := 100 * time.Millisecond

	// Mock empty project list for initial check
	mockProjectService.On("List").Return([]*services.Project{}, nil)

	service := NewWatcherService(mockProjectService, mockGitService, pollInterval)

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	// Start should return nil when context is cancelled
	err := service.Start(ctx)
	assert.NoError(t, err)

	// Verify the initial check was called
	mockProjectService.AssertExpectations(t)
}

func TestWatcherService_checkAllProjects_EmptyList(t *testing.T) {
	mockProjectService := &MockProjectManager{}
	mockGitService := &MockGitExecutor{}
	service := NewWatcherService(mockProjectService, mockGitService, time.Minute)

	mockProjectService.On("List").Return([]*services.Project{}, nil)

	err := service.checkAllProjects(context.Background())
	assert.NoError(t, err)
	mockProjectService.AssertExpectations(t)
}

func TestWatcherService_checkAllProjects_MixedProjects(t *testing.T) {
	mockProjectService := &MockProjectManager{}
	mockGitService := &MockGitExecutor{}
	service := NewWatcherService(mockProjectService, mockGitService, time.Minute)

	projects := []*services.Project{
		createTestProject(uuid.New(), "running-with-watcher", services.ProjectStatusRunning, true, "commit1"),
		createTestProject(uuid.New(), "stopped-with-watcher", services.ProjectStatusStopped, true, "commit2"),
		createTestProject(uuid.New(), "running-without-watcher", services.ProjectStatusRunning, false, "commit3"),
		createTestProject(uuid.New(), "error-with-watcher", services.ProjectStatusError, true, "commit4"),
	}

	mockProjectService.On("List").Return(projects, nil)

	// Mock GetStatus calls for all projects (new functionality)
	runningStatus := &services.ComposeStatus{Status: services.ComposeProjectStatusRunning}
	stoppedStatus := &services.ComposeStatus{Status: services.ComposeProjectStatusStopped}

	mockProjectService.On("GetStatus", projects[0].ID).Return(runningStatus, nil) // running-with-watcher - matches DB
	mockProjectService.On("GetStatus", projects[1].ID).Return(stoppedStatus, nil) // stopped-with-watcher - matches DB
	mockProjectService.On("GetStatus", projects[2].ID).
		Return(runningStatus, nil)
		// running-without-watcher - matches DB
	mockProjectService.On("GetStatus", projects[3].ID).
		Return(stoppedStatus, nil)
		// error-with-watcher - mismatch, should update to stopped

	// Error project status should be updated to stopped since Docker shows stopped
	errorProjectID := projects[3].ID
	mockProjectService.On("Update", mock.MatchedBy(func(p *services.Project) bool {
		return p.ID == errorProjectID && p.Status == services.ProjectStatusStopped
	})).Return(nil)

	// Only the running project with watcher enabled should be checked (stopped projects are excluded)
	mockGitService.On("Fetch", "main", (*services.GitAuthConfig)(nil), "/tmp/test-project-running-with-watcher/git").
		Return(nil)
	mockGitService.On("GetRemoteLatestCommit", "/tmp/test-project-running-with-watcher/git", "main").
		Return("commit1", nil)

	// Note: Error project is no longer checked for git changes because its status was corrected to "stopped"

	err := service.checkAllProjects(context.Background())
	assert.NoError(t, err)

	mockProjectService.AssertExpectations(t)
	mockGitService.AssertExpectations(t)
}

func TestWatcherService_checkProject_NoChanges(t *testing.T) {
	mockProjectService := &MockProjectManager{}
	mockGitService := &MockGitExecutor{}
	service := NewWatcherService(mockProjectService, mockGitService, time.Minute)

	project := createTestProject(uuid.New(), "test-project", services.ProjectStatusRunning, true, "commit1")

	mockGitService.On("Fetch", "main", (*services.GitAuthConfig)(nil), "/tmp/test-project-test-project/git").Return(nil)
	mockGitService.On("GetRemoteLatestCommit", "/tmp/test-project-test-project/git", "main").Return("commit1", nil)

	err := service.checkProject(context.Background(), project)
	assert.NoError(t, err)

	mockGitService.AssertExpectations(t)
	// DeployPiping should not be called since commits are the same
	mockProjectService.AssertNotCalled(t, "DeployPiping")
}

func TestWatcherService_checkProject_ErrorStateWithoutChanges(t *testing.T) {
	mockProjectService := &MockProjectManager{}
	mockGitService := &MockGitExecutor{}
	service := NewWatcherService(mockProjectService, mockGitService, time.Minute)

	project := createTestProject(uuid.New(), "test-project", services.ProjectStatusError, true, "commit1")

	mockGitService.On("Fetch", "main", (*services.GitAuthConfig)(nil), "/tmp/test-project-test-project/git").Return(nil)
	mockGitService.On("GetRemoteLatestCommit", "/tmp/test-project-test-project/git", "main").
		Return("commit1", nil)
		// Same commit
	mockProjectService.On("DeployPiping", project.ID, true).
		Return(nil)
		// Should deploy despite no git changes
	mockProjectService.On("Update", mock.MatchedBy(func(p *services.Project) bool {
		return p.ID == project.ID && p.LastCommit != nil && *p.LastCommit == "commit1"
	})).Return(nil)

	err := service.checkProject(context.Background(), project)
	assert.NoError(t, err)

	mockGitService.AssertExpectations(t)
	mockProjectService.AssertExpectations(t)
}

func TestWatcherService_checkProject_WithChanges(t *testing.T) {
	mockProjectService := &MockProjectManager{}
	mockGitService := &MockGitExecutor{}
	service := NewWatcherService(mockProjectService, mockGitService, time.Minute)

	project := createTestProject(uuid.New(), "test-project", services.ProjectStatusRunning, true, "commit1")

	mockGitService.On("Fetch", "main", (*services.GitAuthConfig)(nil), "/tmp/test-project-test-project/git").Return(nil)
	mockGitService.On("GetRemoteLatestCommit", "/tmp/test-project-test-project/git", "main").Return("commit2", nil)
	mockProjectService.On("DeployPiping", project.ID, true).Return(nil)
	mockProjectService.On("Update", mock.MatchedBy(func(p *services.Project) bool {
		return p.ID == project.ID && p.LastCommit != nil && *p.LastCommit == "commit2"
	})).Return(nil)

	err := service.checkProject(context.Background(), project)
	assert.NoError(t, err)

	mockGitService.AssertExpectations(t)
	mockProjectService.AssertExpectations(t)
}

func TestWatcherService_checkProject_FetchError(t *testing.T) {
	mockProjectService := &MockProjectManager{}
	mockGitService := &MockGitExecutor{}
	service := NewWatcherService(mockProjectService, mockGitService, time.Minute)

	project := createTestProject(uuid.New(), "test-project", services.ProjectStatusRunning, true, "commit1")

	mockGitService.On("Fetch", "main", (*services.GitAuthConfig)(nil), "/tmp/test-project-test-project/git").
		Return(assert.AnError)

	err := service.checkProject(context.Background(), project)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fetch from remote")

	mockGitService.AssertExpectations(t)
	mockProjectService.AssertNotCalled(t, "DeployPiping")
}

func TestWatcherService_checkProject_GetRemoteCommitError(t *testing.T) {
	mockProjectService := &MockProjectManager{}
	mockGitService := &MockGitExecutor{}
	service := NewWatcherService(mockProjectService, mockGitService, time.Minute)

	project := createTestProject(uuid.New(), "test-project", services.ProjectStatusRunning, true, "commit1")

	mockGitService.On("Fetch", "main", (*services.GitAuthConfig)(nil), "/tmp/test-project-test-project/git").Return(nil)
	mockGitService.On("GetRemoteLatestCommit", "/tmp/test-project-test-project/git", "main").Return("", assert.AnError)

	err := service.checkProject(context.Background(), project)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get remote commit")

	mockGitService.AssertExpectations(t)
	mockProjectService.AssertNotCalled(t, "DeployPiping")
}

func TestWatcherService_checkProject_DeploymentError(t *testing.T) {
	mockProjectService := &MockProjectManager{}
	mockGitService := &MockGitExecutor{}
	service := NewWatcherService(mockProjectService, mockGitService, time.Minute)

	project := createTestProject(uuid.New(), "test-project", services.ProjectStatusRunning, true, "commit1")

	mockGitService.On("Fetch", "main", (*services.GitAuthConfig)(nil), "/tmp/test-project-test-project/git").Return(nil)
	mockGitService.On("GetRemoteLatestCommit", "/tmp/test-project-test-project/git", "main").Return("commit2", nil)
	mockProjectService.On("DeployPiping", project.ID, true).Return(assert.AnError)

	err := service.checkProject(context.Background(), project)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to deploy project")

	mockGitService.AssertExpectations(t)
	mockProjectService.AssertExpectations(t)
}

func TestWatcherService_checkProject_UpdateError(t *testing.T) {
	mockProjectService := &MockProjectManager{}
	mockGitService := &MockGitExecutor{}
	service := NewWatcherService(mockProjectService, mockGitService, time.Minute)

	project := createTestProject(uuid.New(), "test-project", services.ProjectStatusRunning, true, "commit1")

	mockGitService.On("Fetch", "main", (*services.GitAuthConfig)(nil), "/tmp/test-project-test-project/git").Return(nil)
	mockGitService.On("GetRemoteLatestCommit", "/tmp/test-project-test-project/git", "main").Return("commit2", nil)
	mockProjectService.On("DeployPiping", project.ID, true).Return(nil)
	mockProjectService.On("Update", mock.MatchedBy(func(p *services.Project) bool {
		return p.ID == project.ID && p.LastCommit != nil && *p.LastCommit == "commit2"
	})).Return(assert.AnError)

	// Should not fail even if update fails - deployment was successful
	err := service.checkProject(context.Background(), project)
	assert.NoError(t, err)

	mockGitService.AssertExpectations(t)
	mockProjectService.AssertExpectations(t)
}

func TestWatcherService_checkAllProjects_ListError(t *testing.T) {
	mockProjectService := &MockProjectManager{}
	mockGitService := &MockGitExecutor{}
	service := NewWatcherService(mockProjectService, mockGitService, time.Minute)

	mockProjectService.On("List").Return([]*services.Project{}, assert.AnError)

	err := service.checkAllProjects(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list projects")

	mockProjectService.AssertExpectations(t)
}

func TestWatcherService_Start_WithPolling(t *testing.T) {
	mockProjectService := &MockProjectManager{}
	mockGitService := &MockGitExecutor{}

	// Use a very short poll interval for testing
	pollInterval := 50 * time.Millisecond
	service := NewWatcherService(mockProjectService, mockGitService, pollInterval)

	// Mock will be called multiple times
	mockProjectService.On("List").Return([]*services.Project{}, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err := service.Start(ctx)
	assert.NoError(t, err)

	// Verify that List was called multiple times (at least 3: initial + 2 polls)
	assert.GreaterOrEqual(t, len(mockProjectService.Calls), 3)
}

func TestWatcherService_checkProject_WithGitAuth(t *testing.T) {
	mockProjectService := &MockProjectManager{}
	mockGitService := &MockGitExecutor{}
	service := NewWatcherService(mockProjectService, mockGitService, time.Minute)

	gitAuth := &services.GitAuthConfig{
		HTTPAuth: &services.GitHTTPAuthConfig{
			Username: "testuser",
			Password: "testpass",
		},
	}

	project := createTestProject(uuid.New(), "test-project", services.ProjectStatusRunning, true, "commit1")
	project.GitAuth = gitAuth

	mockGitService.On("Fetch", "main", gitAuth, "/tmp/test-project-test-project/git").Return(nil)
	mockGitService.On("GetRemoteLatestCommit", "/tmp/test-project-test-project/git", "main").Return("commit1", nil)

	err := service.checkProject(context.Background(), project)
	assert.NoError(t, err)

	mockGitService.AssertExpectations(t)
}

// Test for syncProjectStatus when GetStatus fails - should update database to unknown
func TestWatcherService_syncProjectStatus_GetStatusError(t *testing.T) {
	mockProjectService := &MockProjectManager{}
	mockGitService := &MockGitExecutor{}
	service := NewWatcherService(mockProjectService, mockGitService, time.Minute)

	project := createTestProject(uuid.New(), "test-project", services.ProjectStatusRunning, false, "commit1")

	// Mock GetStatus to return an error
	mockProjectService.On("GetStatus", project.ID).
		Return((*services.ComposeStatus)(nil), errors.New("docker not available"))

	// Expect project status to be updated to unknown
	mockProjectService.On("Update", mock.MatchedBy(func(p *services.Project) bool {
		return p.ID == project.ID && p.Status == services.ProjectStatusUnknown
	})).Return(nil)

	err := service.syncProjectStatus(context.Background(), project)
	assert.Error(t, err, "should return error when GetStatus fails")

	mockProjectService.AssertExpectations(t)
}

// Test for syncProjectStatus with ComposeProjectStatusFailed
func TestWatcherService_syncProjectStatus_FailedStatus(t *testing.T) {
	mockProjectService := &MockProjectManager{}
	mockGitService := &MockGitExecutor{}
	service := NewWatcherService(mockProjectService, mockGitService, time.Minute)

	project := createTestProject(uuid.New(), "test-project", services.ProjectStatusRunning, false, "commit1")

	// Mock GetStatus to return failed status
	failedStatus := &services.ComposeStatus{Status: services.ComposeProjectStatusFailed}
	mockProjectService.On("GetStatus", project.ID).Return(failedStatus, nil)

	// Expect project status to be updated to error (since failed maps to error)
	mockProjectService.On("Update", mock.MatchedBy(func(p *services.Project) bool {
		return p.ID == project.ID && p.Status == services.ProjectStatusError
	})).Return(nil)

	err := service.syncProjectStatus(context.Background(), project)
	assert.NoError(t, err)

	mockProjectService.AssertExpectations(t)
}

// Test for syncProjectStatus with ComposeProjectStatusUnknown
func TestWatcherService_syncProjectStatus_UnknownStatus(t *testing.T) {
	mockProjectService := &MockProjectManager{}
	mockGitService := &MockGitExecutor{}
	service := NewWatcherService(mockProjectService, mockGitService, time.Minute)

	project := createTestProject(uuid.New(), "test-project", services.ProjectStatusRunning, false, "commit1")

	// Mock GetStatus to return unknown status
	unknownStatus := &services.ComposeStatus{Status: services.ComposeProjectStatusUnknown}
	mockProjectService.On("GetStatus", project.ID).Return(unknownStatus, nil)

	// Expect project status to be updated to unknown
	mockProjectService.On("Update", mock.MatchedBy(func(p *services.Project) bool {
		return p.ID == project.ID && p.Status == services.ProjectStatusUnknown
	})).Return(nil)

	err := service.syncProjectStatus(context.Background(), project)
	assert.NoError(t, err)

	mockProjectService.AssertExpectations(t)
}

// Test for syncProjectStatus when status matches (no update needed)
func TestWatcherService_syncProjectStatus_NoUpdateNeeded(t *testing.T) {
	mockProjectService := &MockProjectManager{}
	mockGitService := &MockGitExecutor{}
	service := NewWatcherService(mockProjectService, mockGitService, time.Minute)

	project := createTestProject(uuid.New(), "test-project", services.ProjectStatusRunning, false, "commit1")

	// Mock GetStatus to return same status as project already has
	runningStatus := &services.ComposeStatus{Status: services.ComposeProjectStatusRunning}
	mockProjectService.On("GetStatus", project.ID).Return(runningStatus, nil)

	// No Update call should be made since status matches

	err := service.syncProjectStatus(context.Background(), project)
	assert.NoError(t, err)

	mockProjectService.AssertExpectations(t)
}
