// Package models provides the database models for Oar.
package models

import (
	"time"

	"github.com/google/uuid"
)

type Project struct {
	ID               uuid.UUID `gorm:"type:char(36);primaryKey"`
	Name             string    `gorm:"not null;unique"`
	GitURL           string    `gorm:"not null"`
	ComposeName      string    `gorm:"not null;unique"`
	ComposeFiles     string    `gorm:"not null"` // list of compose file paths separated by null character (\0)
	EnvironmentFiles string    `gorm:"not null"` // list of environment file paths separated by null character (\0)
	LastCommit       *string
	CreatedAt        time.Time
	UpdatedAt        time.Time

	Deployments []Deployment `gorm:"foreignKey:ProjectID"`
}

type Secret struct {
	ID        uuid.UUID `gorm:"type:char(36);primaryKey"`
	ProjectID uuid.UUID `gorm:"not null;index"`
	Name      string    `gorm:"not null"`
	Value     string    `gorm:"not null"` // TODO: Encrypt this field
	CreatedAt time.Time
	UpdatedAt time.Time

	Project Project `gorm:"foreignKey:ProjectID"`
}

type Deployment struct {
	ID         uuid.UUID `gorm:"type:char(36);primaryKey"`
	ProjectID  uuid.UUID `gorm:"not null;index"`
	CommitHash string    `gorm:"not null"`
	Status     string    `gorm:"not null"`  // in_progress, success, failed
	Output     string    `gorm:"type:text"` // Command output/logs
	CreatedAt  time.Time

	Project Project `gorm:"foreignKey:ProjectID"`
}
