package services

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/ch00k/oar/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	homeDir, _   = os.UserHomeDir()
	workspaceDir = filepath.Join(homeDir, ".oar", "projects")
)

type CreateProjectConfig struct {
	GitURL string
	Name   string
}

type DeploymentConfig struct {
	Detach bool
	Build  bool
	Pull   bool
}

// ProjectService provides methods to manage Docker Compose projects.
type ProjectService struct {
	db                   *gorm.DB
	gitService           GitService
	dockerComposeService DockerComposeProjectService
}

func (s *ProjectService) projectWorkingDir(projectID uuid.UUID) string {
	return filepath.Join(workspaceDir, projectID.String())
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
func (s *ProjectService) CreateProject(config CreateProjectConfig) (*models.Project, error) {
	// Generate UUID upfront. We will later use it for clone directory name, and as project ID.
	projectID := uuid.New()

	workingDir := s.projectWorkingDir(projectID)

	// Clone repository first
	if err := s.gitService.Clone(config.GitURL, workingDir); err != nil {
		return nil, fmt.Errorf("failed to clone repository: %w", err)
	}

	// Get commit info
	commit, _ := s.gitService.GetLatestCommit(workingDir)

	// Determine project name
	// TODO: Use `name` from compose.yml if available, otherwise use config.Name
	projectName := config.Name

	// Determine compose file path
	var composeFileName string

	if _, err := os.Stat(filepath.Join(workingDir, "compose.yml")); err == nil {
		composeFileName = "compose.yml"
	} else if _, err := os.Stat(filepath.Join(workingDir, "compose.yaml")); err == nil {
		composeFileName = "compose.yaml"
	} else if _, err := os.Stat(filepath.Join(workingDir, "docker-compose.yml")); err == nil {
		composeFileName = "docker-compose.yml"
	} else if _, err := os.Stat(filepath.Join(workingDir, "docker-compose.yaml")); err == nil {
		composeFileName = "docker-compose.yaml"
	} else {
		return nil, fmt.Errorf("no valid compose file found in repository")
	}

	// Create database model from config
	project := &models.Project{
		ID:              projectID, // Set explicitly
		Name:            projectName,
		GitURL:          config.GitURL,
		ComposeName:     projectName,
		ComposeFileName: composeFileName,
		LastCommit:      &commit,
	}

	if err := s.db.Create(project).Error; err != nil {
		// Cleanup on failure
		os.RemoveAll(workingDir)
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	return project, nil
}

func (s *ProjectService) DeployProject(projectID uuid.UUID, config DeploymentConfig) (*models.Deployment, error) {
	// Get project
	project, err := s.GetProject(projectID)
	if err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}

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
	if config.Pull {
		if err := s.pullLatestChanges(project); err != nil {
			s.updateDeploymentStatus(deploymentID, "failed", fmt.Sprintf("Git pull failed: %v", err))
			return nil, fmt.Errorf("failed to pull latest changes: %w", err)
		}
	}

	// Deploy using Docker Compose
	output, err := s.deployWithDockerCompose(project, config)
	if err != nil {
		s.updateDeploymentStatus(deploymentID, "failed", fmt.Sprintf("Docker Compose failed: %v\nOutput: %s", err, output))
		return nil, fmt.Errorf("deployment failed: %w", err)
	}

	// Update deployment with success
	deployment.Status = "running"
	deployment.Output = output
	deployment.CommitHash, _ = s.gitService.GetLatestCommit(fmt.Sprintf("%s/%s", workspaceDir, project.ID))

	if err := s.db.Save(deployment).Error; err != nil {
		return nil, fmt.Errorf("failed to update deployment record: %w", err)
	}

	// Update project status and commit
	// project.Status = "running"
	project.LastCommit = &deployment.CommitHash
	s.db.Save(project)

	return deployment, nil
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

func (s *ProjectService) deployWithDockerCompose(project *models.Project, config DeploymentConfig) (string, error) {
	slog.Info("Starting Docker Compose deployment",
		"project_id", project.ID,
		"project_name", project.Name,
		"compose_file", project.ComposeFileName,
		"detach", config.Detach,
		"build", config.Build,
		"pull", config.Pull)

	outputStr, err := s.dockerComposeService.Deploy(project.ComposeName, s.projectWorkingDir(project.ID), project.ComposeFileName, config)
	if err != nil {
		slog.Error("Docker Compose deployment failed",
			"project_id", project.ID,
			"error", err,
			"output", outputStr)
		return outputStr, fmt.Errorf("docker compose command failed: %w", err)
	}

	slog.Info("Docker Compose deployment completed", "project_id", project.ID)
	return outputStr, nil
}

func (s *ProjectService) updateDeploymentStatus(deploymentID uuid.UUID, status, output string) {
	s.db.Model(&models.Deployment{}).
		Where("id = ?", deploymentID).
		Updates(map[string]any{
			"status": status,
			"output": output,
		})
}

func NewProjectService(db *gorm.DB) *ProjectService {
	return &ProjectService{
		db:                   db,
		gitService:           NewGitService(),
		dockerComposeService: NewDockerComposeService(),
	}
}
