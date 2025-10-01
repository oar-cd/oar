package project

import (
	"github.com/google/uuid"
	"github.com/oar-cd/oar/docker"
	"github.com/oar-cd/oar/domain"
)

// ProjectManager defines the contract for project management operations
type ProjectManager interface {
	List() ([]*domain.Project, error)
	Get(id uuid.UUID) (*domain.Project, error)
	Create(project *domain.Project) (*domain.Project, error)
	Update(project *domain.Project) error
	Remove(projectID uuid.UUID, removeVolumes bool) error
	DeployStreaming(projectID uuid.UUID, pull bool, outputChan chan<- docker.StreamMessage) error
	DeployPiping(projectID uuid.UUID, pull bool) error
	Stop(projectID uuid.UUID, removeVolumes bool) error
	StopStreaming(projectID uuid.UUID, outputChan chan<- docker.StreamMessage) error
	StopPiping(projectID uuid.UUID) error
	GetLogs(projectID uuid.UUID) (string, string, error)
	GetLogsPiping(projectID uuid.UUID) error
	GetConfig(projectID uuid.UUID) (string, string, error)
	GetStatus(projectID uuid.UUID) (*docker.ComposeStatus, error)
	ListDeployments(projectID uuid.UUID) ([]*domain.Deployment, error)
}
