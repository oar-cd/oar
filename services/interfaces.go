package services

import (
	"github.com/google/uuid"
)

// GitExecutor defines the contract for Git operations
type GitExecutor interface {
	Clone(gitURL string, gitBranch string, gitAuth *GitAuthConfig, workingDir string) error
	Pull(gitBranch string, gitAuth *GitAuthConfig, workingDir string) error
	Fetch(gitBranch string, gitAuth *GitAuthConfig, workingDir string) error
	GetLatestCommit(workingDir string) (string, error)
	GetRemoteLatestCommit(workingDir string, gitBranch string) (string, error)
	TestAuthentication(gitURL string, gitAuth *GitAuthConfig) error
	GetDefaultBranch(gitURL string, gitAuth *GitAuthConfig) (string, error)
}

// ComposeProjectInterface defines the contract for Docker Compose operations
type ComposeProjectInterface interface {
	Up() (string, error)
	Down() (string, error)
	Logs() (string, error)
	GetConfig() (string, error)
	Status() (*ComposeStatus, error)
	UpStreaming(outputChan chan<- string) error
	UpPiping() error
	DownStreaming(outputChan chan<- string) error
	DownPiping() error
	LogsStreaming(outputChan chan<- string) error
	LogsPiping() error
}

// ProjectManager defines the contract for project management operations
type ProjectManager interface {
	List() ([]*Project, error)
	Get(id uuid.UUID) (*Project, error)
	Create(project *Project) (*Project, error)
	Update(project *Project) error
	Remove(projectID uuid.UUID) error
	DeployStreaming(projectID uuid.UUID, pull bool, outputChan chan<- string) error
	DeployPiping(projectID uuid.UUID, pull bool) error
	Start(projectID uuid.UUID) error
	Stop(projectID uuid.UUID) error
	StopStreaming(projectID uuid.UUID, outputChan chan<- string) error
	StopPiping(projectID uuid.UUID) error
	GetLogsStreaming(projectID uuid.UUID, outputChan chan<- string) error
	GetLogsPiping(projectID uuid.UUID) error
	GetConfig(projectID uuid.UUID) (string, error)
	GetStatus(projectID uuid.UUID) (*ComposeStatus, error)
}
