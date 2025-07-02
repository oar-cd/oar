package services

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/ch00k/oar/models"
	"github.com/google/uuid"
)

type Project struct {
	ID               uuid.UUID
	Name             string
	GitURL           string
	WorkingDir       string
	ComposeFiles     []string
	EnvironmentFiles []string
	Status           ProjectStatus
	LastCommit       *string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (p *Project) GitDir() (string, error) {
	if p.WorkingDir == "" {
		return "", fmt.Errorf("working directory is not set for project %s", p.Name)
	}
	return filepath.Join(p.WorkingDir, GitDir), nil
}

func NewProject(name, gitURL string, composeFiles, environmentFiles []string) Project {
	return Project{
		ID:               uuid.New(),
		Name:             name,
		GitURL:           gitURL,
		ComposeFiles:     composeFiles,
		EnvironmentFiles: environmentFiles,
		Status:           ProjectStatusStopped,
	}
}

func NewProjectFromModel(project *models.ProjectModel, baseDir string) Project {
	status, err := ParseProjectStatus(project.Status)
	if err != nil {
		status = ProjectStatusUnknown
	}

	return Project{
		ID:         project.ID,
		Name:       project.Name,
		GitURL:     project.GitURL,
		WorkingDir: project.WorkingDir,
		ComposeFiles: strings.Split(
			project.ComposeFiles,
			"\000",
		), // Split by null character
		EnvironmentFiles: strings.Split(
			project.EnvironmentFiles,
			"\000",
		), // Split by null character
		Status:     status,
		LastCommit: project.LastCommit,
		CreatedAt:  project.CreatedAt,
		UpdatedAt:  project.UpdatedAt,
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
