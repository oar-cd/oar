package services

import (
	"github.com/ch00k/oar/models"
	"github.com/google/uuid"
)

// GitExecutor defines the contract for Git operations
type GitExecutor interface {
	Clone(gitURL, workingDir string) error
	Pull(workingDir string) error
	GetLatestCommit(workingDir string) (string, error)
	GetCommitInfo(workingDir string) (*CommitInfo, error)
}

// DockerComposeExecutor defines the contract for Docker Compose operations
type DockerComposeExecutor interface {
	Deploy(name, workingDir, composeFile string, config DeploymentConfig) (string, error)
	Down(name, workingDir, composeFile string) (string, error)
}

// ProjectManager defines the contract for project management operations
type ProjectManager interface {
	ListProjects() ([]*models.Project, error)
	GetProject(id uuid.UUID) (*models.Project, error)
	CreateProject(config CreateProjectConfig) (*models.Project, error)
	DeployProject(projectID uuid.UUID, config DeploymentConfig) (*models.Deployment, error)
	StopProject(projectID uuid.UUID) error
	RemoveProject(projectID uuid.UUID) error
}