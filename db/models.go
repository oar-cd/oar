// Package db provides database models and utilities for Oar.
package db

import (
	"time"

	"github.com/google/uuid"
)

type BaseModel struct {
	ID        uuid.UUID `gorm:"type:char(36);primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ProjectModel struct {
	BaseModel
	Name               string  `gorm:"not null;unique;check:name <> ''"`
	GitURL             string  `gorm:"not null;check:git_url <> ''"`
	GitBranch          string  `gorm:"not null;check:git_branch <> ''"`    // Git branch (never empty, always set to default branch if not specified)
	GitAuthType        *string `gorm:"type:varchar(20)"`                   // "http", "ssh", "oauth", etc.
	GitAuthCredentials *string `gorm:"type:text"`                          // Encrypted JSON blob containing all auth data
	WorkingDir         string  `gorm:"not null;check:working_dir <> ''"`   // directory where the project is cloned
	ComposeFiles       string  `gorm:"not null;check:compose_files <> ''"` // list of compose file paths separated by null character (\0)
	ComposeOverride    *string `gorm:"type:text"`                          // Optional Docker Compose override content
	Variables          string  `gorm:"not null"`                           // Variables separated by null character (\0)
	Status             string  `gorm:"not null;check:status <> ''"`        // running, stopped, error
	LastCommit         *string
	WatcherEnabled     bool `gorm:"not null"` // Enable automatic deployments on git changes

	Deployments []DeploymentModel `gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE"`
}

func (ProjectModel) TableName() string {
	return "projects"
}

type DeploymentModel struct {
	BaseModel
	ProjectID  uuid.UUID `gorm:"not null;index"`
	CommitHash string    `gorm:"not null;check:commit_hash <> ''"`
	Status     string    `gorm:"not null;check:status <> ''"` // in_progress, success, failed
	Stdout     string    `gorm:"type:text"`                   // Command stdout output
	Stderr     string    `gorm:"type:text"`                   // Command stderr output

	Project ProjectModel `gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE"`
}

func (DeploymentModel) TableName() string {
	return "deployments"
}
