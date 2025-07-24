package services

import "github.com/ch00k/oar/models"

type ProjectMapper struct{}

func (m *ProjectMapper) ToDomain(p *models.ProjectModel) *Project {
	status, err := ParseProjectStatus(p.Status)
	if err != nil {
		status = ProjectStatusUnknown
	}

	return &Project{
		ID:               p.ID,
		Name:             p.Name,
		GitURL:           p.GitURL,
		WorkingDir:       p.WorkingDir,
		ComposeFiles:     parseFiles(p.ComposeFiles),
		EnvironmentFiles: parseFiles(p.EnvironmentFiles),
		Status:           status,
		LastCommit:       p.LastCommit,
		CreatedAt:        p.CreatedAt,
		UpdatedAt:        p.UpdatedAt,
	}
}

func (m *ProjectMapper) ToModel(p *Project) *models.ProjectModel {
	return &models.ProjectModel{
		BaseModel: models.BaseModel{
			ID: p.ID,
		},
		Name:             p.Name,
		GitURL:           p.GitURL,
		WorkingDir:       p.WorkingDir,
		ComposeFiles:     serializeFiles(p.ComposeFiles),
		EnvironmentFiles: serializeFiles(p.EnvironmentFiles),
		Status:           p.Status.String(),
		LastCommit:       p.LastCommit,
	}
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
