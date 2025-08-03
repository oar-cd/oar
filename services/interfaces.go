package services

import (
	"github.com/google/uuid"
)

// GitExecutor defines the contract for Git operations
type GitExecutor interface {
	Clone(gitURL string, gitAuth *GitAuthConfig, workingDir string) error
	Pull(gitAuth *GitAuthConfig, workingDir string) error
	GetLatestCommit(workingDir string) (string, error)
	TestAuthentication(gitURL string, gitAuth *GitAuthConfig) error
}

// ComposeProjectInterface defines the contract for Docker Compose operations
type ComposeProjectInterface interface {
	Up() (string, error)
	Down() (string, error)
	Logs() (string, error)
	UpStreaming(outputChan chan<- string) error
	DownStreaming(outputChan chan<- string) error
	LogsStreaming(outputChan chan<- string) error
}

// ProjectManager defines the contract for project management operations
type ProjectManager interface {
	List() ([]*Project, error)
	Get(id uuid.UUID) (*Project, error)
	GetByName(name string) (*Project, error)
	Create(project *Project) (*Project, error)
	Update(project *Project) error
	Remove(projectID uuid.UUID) error
	DeployStreaming(projectID uuid.UUID, pull bool, outputChan chan<- string) error
	Start(projectID uuid.UUID) error
	StartStreaming(projectID uuid.UUID, outputChan chan<- string) error
	Stop(projectID uuid.UUID) error
	StopStreaming(projectID uuid.UUID, outputChan chan<- string) error
	GetLogsStreaming(projectID uuid.UUID, outputChan chan<- string) error
}
