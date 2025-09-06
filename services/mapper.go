package services

import (
	"log/slog"

	"github.com/ch00k/oar/models"
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
		ID:           p.ID,
		Name:         p.Name,
		GitURL:       p.GitURL,
		GitAuth:      gitAuth,
		WorkingDir:   p.WorkingDir,
		ComposeFiles: parseFiles(p.ComposeFiles),
		Variables:    parseFiles(p.Variables),
		Status:       status,
		LastCommit:   p.LastCommit,
		CreatedAt:    p.CreatedAt,
		UpdatedAt:    p.UpdatedAt,
	}
}

func (m *ProjectMapper) ToModel(p *Project) *models.ProjectModel {
	model := &models.ProjectModel{
		BaseModel: models.BaseModel{
			ID: p.ID,
		},
		Name:         p.Name,
		GitURL:       p.GitURL,
		WorkingDir:   p.WorkingDir,
		ComposeFiles: serializeFiles(p.ComposeFiles),
		Variables:    serializeFiles(p.Variables),
		Status:       p.Status.String(),
		LastCommit:   p.LastCommit,
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
		ID:          d.ID,
		ProjectID:   d.ProjectID,
		CommitHash:  d.CommitHash,
		CommandLine: d.CommandLine,
		Status:      status,
		Output:      d.Output,
	}
}

func (m *DeploymentMapper) ToModel(d *Deployment) *models.DeploymentModel {
	return &models.DeploymentModel{
		BaseModel: models.BaseModel{
			ID: d.ID,
		},
		ProjectID:   d.ProjectID,
		CommitHash:  d.CommitHash,
		CommandLine: d.CommandLine,
		Status:      d.Status.String(),
		Output:      d.Output,
	}
}
