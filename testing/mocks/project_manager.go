package mocks

import (
	"github.com/google/uuid"
	"github.com/oar-cd/oar/services"
)

// MockProjectManager implements the ProjectManager interface for testing
type MockProjectManager struct {
	ListFunc            func() ([]*services.Project, error)
	GetFunc             func(id uuid.UUID) (*services.Project, error)
	CreateFunc          func(project *services.Project) (*services.Project, error)
	UpdateFunc          func(project *services.Project) error
	RemoveFunc          func(projectID uuid.UUID, removeVolumes bool) error
	DeployStreamingFunc func(projectID uuid.UUID, pull bool, outputChan chan<- string) error
	DeployPipingFunc    func(projectID uuid.UUID, pull bool) error
	StopFunc            func(projectID uuid.UUID, removeVolumes bool) error
	StopStreamingFunc   func(projectID uuid.UUID, outputChan chan<- string) error
	StopPipingFunc      func(projectID uuid.UUID) error
	GetLogsPipingFunc   func(projectID uuid.UUID) error
	GetConfigFunc       func(projectID uuid.UUID) (string, error)
	GetLogsFunc         func(projectID uuid.UUID) (string, error)
	GetStatusFunc       func(projectID uuid.UUID) (*services.ComposeStatus, error)
	ListDeploymentsFunc func(projectID uuid.UUID) ([]*services.Deployment, error)
}

func (m *MockProjectManager) List() ([]*services.Project, error) {
	if m.ListFunc != nil {
		return m.ListFunc()
	}
	return []*services.Project{}, nil
}

func (m *MockProjectManager) Get(id uuid.UUID) (*services.Project, error) {
	if m.GetFunc != nil {
		return m.GetFunc(id)
	}
	return &services.Project{ID: id}, nil
}

func (m *MockProjectManager) Create(project *services.Project) (*services.Project, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(project)
	}
	return project, nil
}

func (m *MockProjectManager) Update(project *services.Project) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(project)
	}
	return nil
}

func (m *MockProjectManager) Remove(projectID uuid.UUID, removeVolumes bool) error {
	if m.RemoveFunc != nil {
		return m.RemoveFunc(projectID, removeVolumes)
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

func (m *MockProjectManager) Stop(projectID uuid.UUID, removeVolumes bool) error {
	if m.StopFunc != nil {
		return m.StopFunc(projectID, removeVolumes)
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

func (m *MockProjectManager) GetLogs(projectID uuid.UUID) (string, error) {
	if m.GetLogsFunc != nil {
		return m.GetLogsFunc(projectID)
	}
	return "mock logs output", nil
}

func (m *MockProjectManager) GetStatus(projectID uuid.UUID) (*services.ComposeStatus, error) {
	if m.GetStatusFunc != nil {
		return m.GetStatusFunc(projectID)
	}
	return &services.ComposeStatus{}, nil
}

func (m *MockProjectManager) ListDeployments(projectID uuid.UUID) ([]*services.Deployment, error) {
	if m.ListDeploymentsFunc != nil {
		return m.ListDeploymentsFunc(projectID)
	}
	return []*services.Deployment{}, nil
}
