// Package models provides the database models for Oar.
package models

import (
	"time"

	"github.com/google/uuid"
)

type ProjectModel struct {
	ID               uuid.UUID `gorm:"type:char(36);primaryKey"`
	Name             string    `gorm:"not null;unique"`
	GitURL           string    `gorm:"not null"`
	WorkingDir       string    `gorm:"not null"` // directory where the project is cloned
	ComposeFiles     string    `gorm:"not null"` // list of compose file paths separated by null character (\0)
	EnvironmentFiles string    `gorm:"not null"` // list of environment file paths separated by null character (\0)
	Status           string    `gorm:"not null"` // running, stopped, error
	LastCommit       *string
	CreatedAt        time.Time
	UpdatedAt        time.Time

	Deployments []DeploymentModel `gorm:"foreignKey:ProjectID"`
}

type SecretModel struct {
	ID        uuid.UUID `gorm:"type:char(36);primaryKey"`
	ProjectID uuid.UUID `gorm:"not null;index"`
	Name      string    `gorm:"not null"`
	Value     string    `gorm:"not null"` // TODO: Encrypt this field
	CreatedAt time.Time
	UpdatedAt time.Time

	Project ProjectModel `gorm:"foreignKey:ProjectID"`
}

type DeploymentModel struct {
	ID          uuid.UUID `gorm:"type:char(36);primaryKey"`
	ProjectID   uuid.UUID `gorm:"not null;index"`
	CommitHash  string    `gorm:"not null"`
	CommandLine string    `gorm:"not null"`  // Command executed for deployment
	Status      string    `gorm:"not null"`  // in_progress, success, failed
	Output      string    `gorm:"type:text"` // Command output/logs
	CreatedAt   time.Time

	Project ProjectModel `gorm:"foreignKey:ProjectID"`
}
