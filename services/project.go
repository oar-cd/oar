// Package services provides interfaces and implementations for various services in Oar.
package services

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/ch00k/oar/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProjectConfig struct {
	Name             string
	GitURL           string
	WorkingDir       string
	ComposeFiles     []string
	EnvironmentFiles []string
}

func NewProjectConfig(name, gitURL string, composeFiles, environmentFiles []string) ProjectConfig {
	return ProjectConfig{
		Name:             name,
		GitURL:           gitURL,
		ComposeFiles:     []string{},
		EnvironmentFiles: []string{},
	}
}

func NewProjectConfigFromModel(project *models.Project, baseDir string) ProjectConfig {
	return ProjectConfig{
		Name:             project.Name,
		GitURL:           project.GitURL,
		WorkingDir:       filepath.Join(baseDir, project.ID.String()),
		ComposeFiles:     strings.Split(project.ComposeFiles, "\000"),     // Split by null character
		EnvironmentFiles: strings.Split(project.EnvironmentFiles, "\000"), // Split by null character
	}
}

type DeploymentConfig struct {
	Pull bool
}

func NewDeploymentConfig(pull bool) DeploymentConfig {
	return DeploymentConfig{
		Pull: pull,
	}
}

// ProjectService provides methods to manage Docker Compose projects.
type ProjectService struct {
	db                   *gorm.DB
	gitService           GitExecutor
	dockerComposeService DockerComposeExecutor
	config               *Config
}

// Ensure ProjectService implements ProjectManager
var _ ProjectManager = (*ProjectService)(nil)

func (s *ProjectService) projectWorkingDir(projectID uuid.UUID) string {
	return s.config.ProjectWorkingDir(projectID.String())
}

// ListProjects returns all projects
func (s *ProjectService) ListProjects() ([]*models.Project, error) {
	var projects []*models.Project
	if err := s.db.Find(&projects).Error; err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}
	return projects, nil
}

// GetProject retrieves a project by ID
func (s *ProjectService) GetProject(id uuid.UUID) (*models.Project, error) {
	var project models.Project
	if err := s.db.First(&project, id).Error; err != nil {
		return nil, fmt.Errorf("failed to get project with ID %s: %w", id, err)
	}
	return &project, nil
}

// CreateProject creates a new project
func (s *ProjectService) CreateProject(projectConfig ProjectConfig) (*models.Project, error) {
	// Generate UUID upfront. We will later use it for clone directory name, and as project ID.
	projectID := uuid.New()

	projectConfig.WorkingDir = s.projectWorkingDir(projectID)

	// Clone repository first
	if err := s.gitService.Clone(projectConfig.GitURL, projectConfig.WorkingDir); err != nil {
		return nil, fmt.Errorf("failed to clone repository: %w", err)
	}

	// Get commit info
	commit, _ := s.gitService.GetLatestCommit(projectConfig.WorkingDir)

	// Determine project name
	// TODO: Use `name` from compose.yml if available, otherwise use config.Name
	projectName := projectConfig.Name

	composeFiles := strings.Join(projectConfig.ComposeFiles, "\000") // Use null character as separator

	// Create database model from config
	project := &models.Project{
		ID:           projectID, // Set explicitly
		Name:         projectName,
		GitURL:       projectConfig.GitURL,
		ComposeName:  projectName,
		ComposeFiles: composeFiles,
		LastCommit:   &commit,
	}

	if err := s.db.Create(project).Error; err != nil {
		// Cleanup on failure
		if err := os.RemoveAll(projectConfig.WorkingDir); err != nil {
			slog.Error("Failed to remove project directory after creation failure", "working_dir", projectConfig.WorkingDir, "error", err)
			return nil, fmt.Errorf("failed to create project: %w", err)
		}
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	return project, nil
}

func (s *ProjectService) DeployProject(projectID uuid.UUID, deploymentConfig *DeploymentConfig) (*models.Deployment, error) {
	// Get project
	project, err := s.GetProject(projectID)
	if err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}

	projectConfig := NewProjectConfigFromModel(project, s.projectWorkingDir(project.ID))

	// Generate deployment ID upfront
	deploymentID := uuid.New()

	// Create initial deployment record
	deployment := &models.Deployment{
		ID:        deploymentID,
		ProjectID: projectID,
		Status:    "starting",
		Project:   *project, // Include project data for response
	}

	if err := s.db.Create(deployment).Error; err != nil {
		return nil, fmt.Errorf("failed to create deployment record: %w", err)
	}

	// Pull latest changes if requested
	if deploymentConfig.Pull {
		if err := s.pullLatestChanges(project); err != nil {
			s.updateDeploymentStatus(deploymentID, "failed", fmt.Sprintf("Git pull failed: %v", err))
			return nil, fmt.Errorf("failed to pull latest changes: %w", err)
		}
	}

	// Deploy using Docker Compose
	slog.Info("Starting Docker Compose deployment",
		"project_id", project.ID,
		"project_name", project.Name,
		"compose_files", project.ComposeFiles,
		"pull", deploymentConfig.Pull)

	output, err := s.dockerComposeService.Up(&projectConfig, deploymentConfig)
	if err != nil {
		slog.Error("Docker Compose deployment failed",
			"project_id", project.ID,
			"error", err,
			"output", output)
		s.updateDeploymentStatus(deploymentID, "failed", fmt.Sprintf("Docker Compose failed: %v\nOutput: %s", err, output))
		return nil, fmt.Errorf("docker compose command failed: %w", err)
	}

	slog.Info("Docker Compose deployment completed", "project_id", project.ID)

	// Update deployment with success
	deployment.Status = "running"
	deployment.Output = output
	deployment.CommitHash, _ = s.gitService.GetLatestCommit(s.projectWorkingDir(project.ID))

	if err := s.db.Save(deployment).Error; err != nil {
		return nil, fmt.Errorf("failed to update deployment record: %w", err)
	}

	// Update project status and commit
	// project.Status = "running"
	project.LastCommit = &deployment.CommitHash
	s.db.Save(project)

	return deployment, nil
}

func (s *ProjectService) StopProject(projectID uuid.UUID) error {
	// Get project
	project, err := s.GetProject(projectID)
	if err != nil {
		return fmt.Errorf("project not found: %w", err)
	}

	projectConfig := NewProjectConfigFromModel(project, s.projectWorkingDir(project.ID))

	// Stop Docker Compose
	slog.Info("Stopping Docker Compose project", "project_id", project.ID, "project_name", project.Name)
	output, err := s.dockerComposeService.Down(&projectConfig)
	if err != nil {
		slog.Error("Docker Compose down failed", "project_id", project.ID, "error", err, "output", output)
		return fmt.Errorf("failed to stop project: %w", err)
	}
	slog.Info("Docker Compose project stopped", "project_id", project.ID, "output_length", len(output))

	if err := s.db.Save(project).Error; err != nil {
		return fmt.Errorf("failed to update project status: %w", err)
	}

	return nil
}

func (s *ProjectService) RemoveProject(projectID uuid.UUID) error {
	// Get project
	project, err := s.GetProject(projectID)
	if err != nil {
		return fmt.Errorf("project not found: %w", err)
	}

	// Stop Docker Compose project if running
	if err := s.StopProject(projectID); err != nil {
		slog.Warn("Failed to stop project before removal", "project_id", project.ID, "error", err)
		return fmt.Errorf("failed to stop project before removal: %w", err)
	}

	// Remove project directory
	workingDir := s.projectWorkingDir(project.ID)
	if err := os.RemoveAll(workingDir); err != nil {
		return fmt.Errorf("failed to remove project directory: %w", err)
	}

	// Delete project from database
	if err := s.db.Delete(&models.Project{}, projectID).Error; err != nil {
		return fmt.Errorf("failed to delete project from database: %w", err)
	}

	slog.Info("Project removed successfully", "project_id", project.ID, "working_dir", workingDir)
	return nil
}

func (s *ProjectService) pullLatestChanges(project *models.Project) error {
	slog.Info("Pulling latest changes", "project_id", project.ID, "git_url", project.GitURL)

	err := s.gitService.Pull(s.projectWorkingDir(project.ID))
	if err != nil {
		slog.Error("Failed to pull changes", "project_id", project.ID, "error", err)
		return fmt.Errorf("failed to pull changes: %w", err)
	}

	slog.Info("Git pull completed", "project_id", project.ID)
	return nil
}

func (s *ProjectService) updateDeploymentStatus(deploymentID uuid.UUID, status, output string) {
	s.db.Model(&models.Deployment{}).
		Where("id = ?", deploymentID).
		Updates(map[string]any{
			"status": status,
			"output": output,
		})
}

// NewProjectService creates a new ProjectService with dependency injection
func NewProjectService(
	db *gorm.DB,
	gitService GitExecutor,
	dockerComposeService DockerComposeExecutor,
	config *Config,
) *ProjectService {
	return &ProjectService{
		db:                   db,
		gitService:           gitService,
		dockerComposeService: dockerComposeService,
		config:               config,
	}
}

// NewProjectServiceWithDefaults creates a ProjectService with default implementations
func NewProjectServiceWithDefaults(db *gorm.DB, config *Config) *ProjectService {
	return NewProjectService(
		db,
		&GitService{},
		&DockerComposeProjectService{},
		config,
	)
}
