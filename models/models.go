// Package models provides the database models for Oar.
package models

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
	Name               string  `gorm:"not null;unique"`
	GitURL             string  `gorm:"not null"`
	GitAuthType        *string `gorm:"type:varchar(20)"` // "http", "ssh", "oauth", etc.
	GitAuthCredentials *string `gorm:"type:text"`        // Encrypted JSON blob containing all auth data
	WorkingDir         string  `gorm:"not null"`         // directory where the project is cloned
	ComposeFiles       string  `gorm:"not null"`         // list of compose file paths separated by null character (\0)
	Variables          string  `gorm:"not null"`         // Variables separated by null character (\0)
	Status             string  `gorm:"not null"`         // running, stopped, error
	LastCommit         *string

	Deployments []DeploymentModel `gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE"`
}

func (ProjectModel) TableName() string {
	return "projects"
}

type DeploymentModel struct {
	BaseModel
	ProjectID   uuid.UUID `gorm:"not null;index"`
	CommitHash  string    `gorm:"not null"`
	CommandLine string    `gorm:"not null"`  // Command executed for deployment
	Status      string    `gorm:"not null"`  // in_progress, success, failed
	Output      string    `gorm:"type:text"` // Command output/logs

	Project ProjectModel `gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE"`
}

func (DeploymentModel) TableName() string {
	return "deployments"
}
