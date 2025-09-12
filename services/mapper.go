package services

import (
	"log/slog"

	"github.com/oar-cd/oar/models"
)

type ProjectMapper struct {
	encryption *EncryptionService
}

func NewProjectMapper(encryption *EncryptionService) *ProjectMapper {
	return &ProjectMapper{encryption: encryption}
}

func (m *ProjectMapper) ToDomain(p *models.ProjectModel) *Project {
	status, err := ParseProjectStatus(p.Status)
	if err != nil {
		status = ProjectStatusUnknown
	}

	// Decrypt authentication data if present
	var gitAuth *GitAuthConfig
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

	return &Project{
		ID:             p.ID,
		Name:           p.Name,
		GitURL:         p.GitURL,
		GitBranch:      p.GitBranch,
		GitAuth:        gitAuth,
		WorkingDir:     p.WorkingDir,
		ComposeFiles:   parseFiles(p.ComposeFiles),
		Variables:      parseFiles(p.Variables),
		Status:         status,
		LastCommit:     p.LastCommit,
		WatcherEnabled: p.WatcherEnabled,
		CreatedAt:      p.CreatedAt,
		UpdatedAt:      p.UpdatedAt,
	}
}

func (m *ProjectMapper) ToModel(p *Project) *models.ProjectModel {
	model := &models.ProjectModel{
		BaseModel: models.BaseModel{
			ID:        p.ID,
			CreatedAt: p.CreatedAt,
			UpdatedAt: p.UpdatedAt,
		},
		Name:           p.Name,
		GitURL:         p.GitURL,
		GitBranch:      p.GitBranch,
		WorkingDir:     p.WorkingDir,
		ComposeFiles:   serializeFiles(p.ComposeFiles),
		Variables:      serializeFiles(p.Variables),
		Status:         p.Status.String(),
		LastCommit:     p.LastCommit,
		WatcherEnabled: p.WatcherEnabled,
	}

	// Encrypt authentication data if present
	if p.GitAuth != nil && m.encryption != nil {
		authType, encryptedCredentials, err := m.encryption.EncryptGitAuthConfig(p.GitAuth)
		if err != nil {
			// For now, we'll skip encryption on error
			// In production, this should be handled more carefully
			return model
		}

		if authType != "" && encryptedCredentials != "" {
			model.GitAuthType = &authType
			model.GitAuthCredentials = &encryptedCredentials
		}
	}

	return model
}

type DeploymentMapper struct{}

func (m *DeploymentMapper) ToDomain(d *models.DeploymentModel) *Deployment {
	status, err := ParseDeploymentStatus(d.Status)
	if err != nil {
		status = DeploymentStatusUnknown
	}

	return &Deployment{
		ID:         d.ID,
		ProjectID:  d.ProjectID,
		CommitHash: d.CommitHash,
		Status:     status,
		Output:     d.Output,
		CreatedAt:  d.CreatedAt,
		UpdatedAt:  d.UpdatedAt,
	}
}

func (m *DeploymentMapper) ToModel(d *Deployment) *models.DeploymentModel {
	return &models.DeploymentModel{
		BaseModel: models.BaseModel{
			ID:        d.ID,
			CreatedAt: d.CreatedAt,
			UpdatedAt: d.UpdatedAt,
		},
		ProjectID:  d.ProjectID,
		CommitHash: d.CommitHash,
		Status:     d.Status.String(),
		Output:     d.Output,
	}
}
