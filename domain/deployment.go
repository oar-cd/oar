package domain

import (
	"time"

	"github.com/google/uuid"
)

type Deployment struct {
	ID         uuid.UUID
	ProjectID  uuid.UUID
	CommitHash string
	Status     DeploymentStatus
	Stdout     string
	Stderr     string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func NewDeployment(projectID uuid.UUID, commitHash string) Deployment {
	return Deployment{
		ID:         uuid.New(),
		ProjectID:  projectID,
		CommitHash: commitHash,
	}
}

type DeploymentResult struct {
	Status DeploymentStatus
	Stdout string
	Stderr string
}
