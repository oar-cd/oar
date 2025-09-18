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
	Up(startServices bool) (string, error)
	Down(removeVolumes bool) (string, error)
	Logs() (string, error)
	GetConfig() (string, error)
	Pull() (string, error)
	Build() (string, error)
	InitializeVolumeMounts() error
	Status() (*ComposeStatus, error)
	UpStreaming(startServices bool, outputChan chan<- string) error
	UpPiping(startServices bool) error
	DownStreaming(outputChan chan<- string) error
	DownPiping() error
	LogsPiping() error
}

// ProjectManager defines the contract for project management operations
type ProjectManager interface {
	List() ([]*Project, error)
	Get(id uuid.UUID) (*Project, error)
	Create(project *Project) (*Project, error)
	Update(project *Project) error
	Remove(projectID uuid.UUID, removeVolumes bool) error
	DeployStreaming(projectID uuid.UUID, pull bool, outputChan chan<- string) error
	DeployPiping(projectID uuid.UUID, pull bool) error
	Stop(projectID uuid.UUID, removeVolumes bool) error
	StopStreaming(projectID uuid.UUID, outputChan chan<- string) error
	StopPiping(projectID uuid.UUID) error
	GetLogs(projectID uuid.UUID) (string, error)
	GetLogsPiping(projectID uuid.UUID) error
	GetConfig(projectID uuid.UUID) (string, error)
	GetStatus(projectID uuid.UUID) (*ComposeStatus, error)
	ListDeployments(projectID uuid.UUID) ([]*Deployment, error)
}
