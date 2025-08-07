package services

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

type Project struct {
	ID           uuid.UUID
	Name         string
	GitURL       string
	GitAuth      *GitAuthConfig // Git authentication configuration
	WorkingDir   string
	ComposeFiles []string
	Variables    []string // Variables in .env format, one per string
	Status       ProjectStatus
	LastCommit   *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (p *Project) GitDir() (string, error) {
	if p.WorkingDir == "" {
		return "", fmt.Errorf("working directory is not set for project %s", p.Name)
	}
	return filepath.Join(p.WorkingDir, GitDir), nil
}

func (p *Project) LastCommitStr() string {
	if p.LastCommit == nil {
		return ""
	}
	return *p.LastCommit
}

func NewProject(name, gitURL string, composeFiles []string, variables []string) Project {
	return Project{
		ID:           uuid.New(),
		Name:         name,
		GitURL:       gitURL,
		ComposeFiles: composeFiles,
		Variables:    variables,
		Status:       ProjectStatusStopped,
	}
}

type Deployment struct {
	ID          uuid.UUID
	ProjectID   uuid.UUID
	CommitHash  string
	CommandLine string
	Status      DeploymentStatus
	Output      string
}

func NewDeployment(projectID uuid.UUID, commitHash string) Deployment {
	return Deployment{
		ID:         uuid.New(),
		ProjectID:  projectID,
		CommitHash: commitHash,
	}
}

type DeploymentResult struct {
	CommandLine string
	Status      DeploymentStatus
	Output      string
}
