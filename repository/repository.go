package repository

import (
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/oar-cd/oar/db"
	"github.com/oar-cd/oar/domain"
	"github.com/oar-cd/oar/encryption"
	"gorm.io/gorm"
)

type ProjectRepository interface {
	FindByID(id uuid.UUID) (*domain.Project, error)
	FindByName(name string) (*domain.Project, error)
	Create(project *domain.Project) (*domain.Project, error)
	Update(project *domain.Project) error
	List() ([]*domain.Project, error)
	Delete(id uuid.UUID) error
}

type projectRepository struct {
	db     *gorm.DB
	mapper *ProjectMapper
}

func (r *projectRepository) List() ([]*domain.Project, error) {
	var models []db.ProjectModel
	if err := r.db.Find(&models).Error; err != nil {
		return nil, err
	}

	projects := make([]*domain.Project, len(models))
	for i, model := range models {
		projects[i] = r.mapper.ToDomain(&model)
	}
	return projects, nil
}

func (r *projectRepository) FindByID(id uuid.UUID) (*domain.Project, error) {
	var m db.ProjectModel
	if err := r.db.First(&m, id).Error; err != nil {
		slog.Error("Database operation failed",
			"layer", "repository",
			"operation", "find_project",
			"project_id", id,
			"error", err)
		return nil, err // Pass through as-is
	}
	return r.mapper.ToDomain(&m), nil
}

func (r *projectRepository) FindByName(name string) (*domain.Project, error) {
	var m db.ProjectModel
	if err := r.db.Where("name = ?", name).First(&m).Error; err != nil {
		return nil, err
	}
	return r.mapper.ToDomain(&m), nil
}

func (r *projectRepository) Create(project *domain.Project) (*domain.Project, error) {
	m := r.mapper.ToModel(project)
	res := r.db.Create(m)
	if res.Error != nil {
		slog.Error("Database operation failed",
			"layer", "repository",
			"operation", "create_project",
			"project_id", project.ID,
			"project_name", project.Name,
			"error", res.Error)
		return nil, res.Error // Pass through as-is
	}
	return r.mapper.ToDomain(m), nil
}

func (r *projectRepository) Update(project *domain.Project) error {
	m := r.mapper.ToModel(project)

	// Use Select to explicitly update all fields except CreatedAt, including empty strings
	// This ensures that clearing variables (empty string) actually updates the database
	// CreatedAt should never be updated after initial creation
	return r.db.Model(&db.ProjectModel{}).
		Where("id = ?", m.ID).
		Select("*").
		Omit("created_at").
		Updates(m).
		Error
}

func (r *projectRepository) Delete(id uuid.UUID) error {
	err := r.db.Delete(&db.ProjectModel{}, id).Error
	if err != nil {
		slog.Error("Database operation failed",
			"layer", "repository",
			"operation", "delete_project",
			"project_id", id,
			"error", err)
	}
	return err // Pass through as-is
}

func NewProjectRepository(db *gorm.DB, encryptionSvc *encryption.EncryptionService) ProjectRepository {
	return &projectRepository{
		db:     db,
		mapper: NewProjectMapper(encryptionSvc),
	}
}

type DeploymentRepository interface {
	FindByID(id uuid.UUID) (*domain.Deployment, error)
	Create(deployment *domain.Deployment) error
	Update(deployment *domain.Deployment) error
	ListByProjectID(projectID uuid.UUID) ([]*domain.Deployment, error)
}

type deploymentRepository struct {
	db     *gorm.DB
	mapper *DeploymentMapper
}

func (r *deploymentRepository) FindByID(id uuid.UUID) (*domain.Deployment, error) {
	var m db.DeploymentModel
	if err := r.db.First(&m, id).Error; err != nil {
		return nil, err
	}
	return r.mapper.ToDomain(&m), nil
}

func (r *deploymentRepository) Create(deployment *domain.Deployment) error {
	m := r.mapper.ToModel(deployment)
	if err := r.db.Create(m).Error; err != nil {
		return err
	}
	// Update the domain object with the timestamps that GORM populated
	*deployment = *r.mapper.ToDomain(m)
	return nil
}

func (r *deploymentRepository) Update(deployment *domain.Deployment) error {
	m := r.mapper.ToModel(deployment)
	if err := r.db.Save(m).Error; err != nil {
		return err
	}
	// Update the domain object with the timestamps that GORM populated
	*deployment = *r.mapper.ToDomain(m)
	return nil
}

func (r *deploymentRepository) ListByProjectID(projectID uuid.UUID) ([]*domain.Deployment, error) {
	var models []db.DeploymentModel
	if err := r.db.Where("project_id = ?", projectID).Order("created_at DESC").Find(&models).Error; err != nil {
		return nil, err
	}

	deployments := make([]*domain.Deployment, len(models))
	for i, m := range models {
		deployments[i] = r.mapper.ToDomain(&m)
	}
	return deployments, nil
}

func NewDeploymentRepository(db *gorm.DB) DeploymentRepository {
	return &deploymentRepository{
		db:     db,
		mapper: &DeploymentMapper{},
	}
}

// Helper functions
func parseFiles(s string) []string {
	if s == "" {
		return []string{}
	}
	return strings.Split(s, "\x00") // null-separated for better handling
}

func serializeFiles(files []string) string {
	return strings.Join(files, "\x00")
}
