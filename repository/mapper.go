// Package repository provides data access layer for projects and deployments.
package repository

import (
	"log/slog"

	"github.com/oar-cd/oar/db"
	"github.com/oar-cd/oar/domain"
	"github.com/oar-cd/oar/encryption"
)

type ProjectMapper struct {
	encryption *encryption.EncryptionService
}

func NewProjectMapper(encryptionSvc *encryption.EncryptionService) *ProjectMapper {
	return &ProjectMapper{encryption: encryptionSvc}
}

func (m *ProjectMapper) ToDomain(p *db.ProjectModel) *domain.Project {
	status, err := domain.ParseProjectStatus(p.Status)
	if err != nil {
		status = domain.ProjectStatusUnknown
	}

	// Decrypt authentication data if present
	var gitAuth *domain.GitAuthConfig
	if p.GitAuthType != nil && p.GitAuthCredentials != nil && m.encryption != nil {
		decryptedAuth, err := m.encryption.DecryptGitAuthConfig(*p.GitAuthType, *p.GitAuthCredentials)
		if err != nil {
			// Log error but don't fail - project should still be usable
			// This could happen if encryption key changed
			slog.Error("Failed to decrypt Git authentication",
				"project_id", p.ID,
				"project_name", p.Name,
				"auth_type", *p.GitAuthType,
				"error", err)
			gitAuth = nil
		} else {
			gitAuth = decryptedAuth
		}
	}

	return &domain.Project{
		ID:                p.ID,
		Name:              p.Name,
		GitURL:            p.GitURL,
		GitBranch:         p.GitBranch,
		GitAuth:           gitAuth,
		WorkingDir:        p.WorkingDir,
		ComposeFiles:      parseFiles(p.ComposeFiles),
		ComposeOverride:   p.ComposeOverride,
		Variables:         parseFiles(p.Variables),
		Status:            status,
		LocalCommit:       p.LocalCommit,
		RemoteCommit:      p.RemoteCommit,
		AutoDeployEnabled: p.AutoDeployEnabled,
		CreatedAt:         p.CreatedAt,
		UpdatedAt:         p.UpdatedAt,
	}
}

func (m *ProjectMapper) ToModel(p *domain.Project) *db.ProjectModel {
	modelObj := &db.ProjectModel{
		BaseModel: db.BaseModel{
			ID:        p.ID,
			CreatedAt: p.CreatedAt,
			UpdatedAt: p.UpdatedAt,
		},
		Name:              p.Name,
		GitURL:            p.GitURL,
		GitBranch:         p.GitBranch,
		WorkingDir:        p.WorkingDir,
		ComposeFiles:      serializeFiles(p.ComposeFiles),
		ComposeOverride:   p.ComposeOverride,
		Variables:         serializeFiles(p.Variables),
		Status:            p.Status.String(),
		LocalCommit:       p.LocalCommit,
		RemoteCommit:      p.RemoteCommit,
		AutoDeployEnabled: p.AutoDeployEnabled,
	}

	// Encrypt authentication data if present
	if p.GitAuth != nil && m.encryption != nil {
		authType, encryptedCredentials, err := m.encryption.EncryptGitAuthConfig(p.GitAuth)
		if err != nil {
			// For now, we'll skip encryption on error
			// In production, this should be handled more carefully
			return modelObj
		}

		if authType != "" && encryptedCredentials != "" {
			modelObj.GitAuthType = &authType
			modelObj.GitAuthCredentials = &encryptedCredentials
		}
	}

	return modelObj
}

type DeploymentMapper struct{}

func (m *DeploymentMapper) ToDomain(d *db.DeploymentModel) *domain.Deployment {
	status, err := domain.ParseDeploymentStatus(d.Status)
	if err != nil {
		status = domain.DeploymentStatusUnknown
	}

	return &domain.Deployment{
		ID:         d.ID,
		ProjectID:  d.ProjectID,
		CommitHash: d.CommitHash,
		Status:     status,
		Stdout:     d.Stdout,
		Stderr:     d.Stderr,
		CreatedAt:  d.CreatedAt,
		UpdatedAt:  d.UpdatedAt,
	}
}

func (m *DeploymentMapper) ToModel(d *domain.Deployment) *db.DeploymentModel {
	return &db.DeploymentModel{
		BaseModel: db.BaseModel{
			ID:        d.ID,
			CreatedAt: d.CreatedAt,
			UpdatedAt: d.UpdatedAt,
		},
		ProjectID:  d.ProjectID,
		CommitHash: d.CommitHash,
		Status:     d.Status.String(),
		Stdout:     d.Stdout,
		Stderr:     d.Stderr,
	}
}
