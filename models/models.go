// Package models provides the database models for Oar.
package models

import (
	"time"

	"github.com/google/uuid"
)

type Deployment struct {
	ID         uuid.UUID `gorm:"type:char(36);primaryKey"`
	ProjectID  uuid.UUID `gorm:"not null;index"`
	CommitHash string    `gorm:"not null"`
	Status     string    `gorm:"not null"`  // in_progress, success, failed
	Output     string    `gorm:"type:text"` // Command output/logs
	CreatedAt  time.Time

	// Relationship
	Project Project `gorm:"foreignKey:ProjectID"`
}

type Project struct {
	ID              uuid.UUID `gorm:"type:char(36);primaryKey"`
	Name            string    `gorm:"not null;unique"`
	GitURL          string    `gorm:"not null"`
	ComposeName     string    `gorm:"not null;unique"`
	ComposeFileName string    `gorm:"not null"`
	LastCommit      *string
	CreatedAt       time.Time
	UpdatedAt       time.Time

	Deployments []Deployment `gorm:"foreignKey:ProjectID"`
}
