// Package project provides project management services for Oar.
package project

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/gosimple/slug"
	"github.com/oar-cd/oar/config"
	"github.com/oar-cd/oar/docker"
	"github.com/oar-cd/oar/domain"
	"github.com/oar-cd/oar/git"
	"github.com/oar-cd/oar/repository"
)

// ProjectService provides methods to manage Docker Compose projects.
type ProjectService struct {
	projectRepository    repository.ProjectRepository
	deploymentRepository repository.DeploymentRepository
	gitService           *git.GitService
	config               *config.Config
}

// Ensure ProjectService implements ProjectManager
var _ ProjectManager = (*ProjectService)(nil)

// List returns all projects
func (s *ProjectService) List() ([]*domain.Project, error) {
	projects, err := s.projectRepository.List()
	if err != nil {
		slog.Error("Service operation failed",
			"layer", "service",
			"operation", "list_projects",
			"error", err)
		return nil, err
	}
	return projects, nil
}

// Get retrieves a project by ID
func (s *ProjectService) Get(id uuid.UUID) (*domain.Project, error) {
	project, err := s.projectRepository.FindByID(id)
	if err != nil {
		slog.Error("Service operation failed",
			"layer", "service",
			"operation", "get_project",
			"project_id", id,
			"error", err)
		return nil, err // Pass through as-is
	}
	return project, nil
}

// Create creates a new project
func (s *ProjectService) Create(project *domain.Project) (*domain.Project, error) {
	// Validate required fields
	if strings.TrimSpace(project.Name) == "" {
		return nil, fmt.Errorf("name is required")
	}
	if strings.TrimSpace(project.GitURL) == "" {
		return nil, fmt.Errorf("git URL is required")
	}
	if len(project.ComposeFiles) == 0 {
		return nil, fmt.Errorf("compose files are required")
	}

	// Create directory name: <project_id>-<normalized_project_name>
	normalizedName := slug.Make(project.Name)
	dirName := fmt.Sprintf("%s-%s", project.ID.String(), normalizedName)
	project.WorkingDir = filepath.Join(s.config.WorkspaceDir, dirName)

	gitDir, err := project.GitDir()
	if err != nil {
		slog.Error("Service operation failed",
			"layer", "service",
			"operation", "create_project",
			"project_id", project.ID,
			"project_name", project.Name,
			"error", err)
		return nil, err
	}

	// Detect default branch if none specified
	if project.GitBranch == "" {
		defaultBranch, err := s.gitService.GetDefaultBranch(project.GitURL, project.GitAuth)
		if err != nil {
			slog.Error("Service operation failed",
				"layer", "service",
				"operation", "create_project_get_default_branch",
				"project_id", project.ID,
				"project_name", project.Name,
				"git_url", project.GitURL,
				"error", err)
			return nil, fmt.Errorf("failed to determine default branch: %w", err)
		}
		project.GitBranch = defaultBranch
		slog.Info(
			"Using detected default branch",
			"project_id",
			project.ID,
			"git_url",
			project.GitURL,
			"default_branch",
			defaultBranch,
		)
	}

	// Clone repository
	if err := s.gitService.Clone(project.GitURL, project.GitBranch, project.GitAuth, gitDir); err != nil {
		slog.Error("Service operation failed",
			"layer", "service",
			"operation", "create_project",
			"project_id", project.ID,
			"project_name", project.Name,
			"git_url", project.GitURL,
			"error", err)
		return nil, err
	}

	// Get commit info
	commit, _ := s.gitService.GetLatestCommit(gitDir)
	project.LastCommit = &commit

	// Set initial status to stopped
	project.Status = domain.ProjectStatusStopped

	// Save working directory for cleanup before repository call
	workingDir := project.WorkingDir

	createdProject, err := s.projectRepository.Create(project)
	if err != nil {
		// Cleanup on failure using saved working directory
		if cleanupErr := os.RemoveAll(workingDir); cleanupErr != nil {
			slog.Error(
				"Failed to remove project directory after creation failure",
				"working_dir",
				workingDir,
				"error",
				cleanupErr,
			)
		}
		slog.Error("Service operation failed",
			"layer", "service",
			"operation", "create_project",
			"project_id", project.ID,
			"project_name", project.Name,
			"git_url", project.GitURL,
			"error", err)
		return nil, err // Pass through as-is
	}

	return createdProject, nil
}

func (s *ProjectService) Update(project *domain.Project) error {
	// Validate required fields
	if strings.TrimSpace(project.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(project.GitURL) == "" {
		return fmt.Errorf("git URL is required")
	}
	if len(project.ComposeFiles) == 0 {
		return fmt.Errorf("compose files are required")
	}
	return s.projectRepository.Update(project)
}

func (s *ProjectService) DeployStreaming(
	projectID uuid.UUID,
	pull bool,
	outputChan chan<- docker.StreamMessage,
) error {
	project, commitHash, deployment, composeProject, err := s.prepareDeployment(projectID, pull)
	if err != nil {
		return err
	}

	// Create buffers to capture stdout and stderr for the deployment record
	var stdoutBuffer, stderrBuffer strings.Builder

	// Helper function to send StreamMessage for web UI only (not stored in DB)
	sendMessage := func(msg, msgType string) {
		outputChan <- docker.StreamMessage{Type: msgType, Content: msg}
	}

	// Streaming-specific messages
	if pull {
		sendMessage("Pulling latest changes from Git...", "info")

		// Get commit hash before pull
		gitDir, err := project.GitDir()
		if err != nil {
			errMsg := fmt.Sprintf("Failed to get git directory: %v", err)
			sendMessage(errMsg, "error")
			return err
		}

		beforeCommit, err := s.gitService.GetLatestCommit(gitDir)
		if err != nil {
			slog.Warn("Failed to get commit hash before pull", "project_id", project.ID, "error", err)
			beforeCommit = "unknown"
		}

		if err := s.pullLatestChanges(project); err != nil {
			errMsg := fmt.Sprintf("Failed to pull latest changes: %v", err)
			sendMessage(errMsg, "error")
			return err
		}

		// Get commit hash after pull
		afterCommit, err := s.gitService.GetLatestCommit(gitDir)
		if err != nil {
			slog.Warn("Failed to get commit hash after pull", "project_id", project.ID, "error", err)
			afterCommit = "unknown"
		}

		// Format commit hashes (use first 8 characters, or full string if "unknown")
		beforeHash := beforeCommit
		if beforeCommit != "unknown" {
			beforeHash = beforeCommit[:8]
		}
		afterHash := afterCommit
		if afterCommit != "unknown" {
			afterHash = afterCommit[:8]
		}

		successMsg := fmt.Sprintf("Git pull completed successfully (from %s to %s)", beforeHash, afterHash)
		sendMessage(successMsg, "success")
	}

	sendMessage("Starting Docker Compose deployment...", "info")

	// Create a capturing channel that forwards Docker stdout/stderr and stores for database
	capturingChan := make(chan docker.StreamMessage, 100)
	done := make(chan bool)

	go func() {
		defer func() { done <- true }()
		for msg := range capturingChan {
			// Store Docker output in appropriate buffer for database
			switch msg.Type {
			case "stdout":
				stdoutBuffer.WriteString(msg.Content + "\n")
			case "stderr":
				stderrBuffer.WriteString(msg.Content + "\n")
			}
			// Forward message directly to user
			outputChan <- msg
		}
	}()

	// Create volumes and containers without starting services
	sendMessage("Creating containers...", "info")
	err = composeProject.UpStreaming(false, capturingChan)
	if err != nil {
		// Ensure we capture any output that was generated before failure
		close(capturingChan)
		<-done
		deployment.Stdout = stdoutBuffer.String()
		deployment.Stderr = stderrBuffer.String()

		errMsg := fmt.Sprintf("Failed to create containers: %v", err)
		sendMessage(errMsg, "error")
		deployment.Status = domain.DeploymentStatusFailed
		if updateErr := s.deploymentRepository.Update(&deployment); updateErr != nil {
			slog.Error("Failed to update deployment status", "error", updateErr)
		}
		return fmt.Errorf("container creation failed: %w", err)
	}

	// Initialize volume permissions
	sendMessage("Initializing volume mounts...", "info")
	if err := composeProject.InitializeVolumeMounts(); err != nil {
		// Ensure we capture any output that was generated before failure
		close(capturingChan)
		<-done
		deployment.Stdout = stdoutBuffer.String()
		deployment.Stderr = stderrBuffer.String()

		errMsg := fmt.Sprintf("Failed to initialize volume permissions: %v", err)
		sendMessage(errMsg, "error")
		deployment.Status = domain.DeploymentStatusFailed
		if updateErr := s.deploymentRepository.Update(&deployment); updateErr != nil {
			slog.Error("Failed to update deployment status", "error", updateErr)
		}
		return fmt.Errorf("volume initialization failed: %w", err)
	}

	// Start services with streaming
	sendMessage("Starting services...", "info")
	err = composeProject.UpStreaming(true, capturingChan)
	close(capturingChan) // Signal that we're done sending to the capturing channel
	<-done               // Wait for the goroutine to finish processing all messages

	// Store the captured stdout and stderr in the deployment record
	deployment.Stdout = stdoutBuffer.String()
	deployment.Stderr = stderrBuffer.String()

	if err != nil {
		return s.handleDeploymentError(project, &deployment, err)
	}

	// Complete deployment
	if err := s.completeDeployment(project, commitHash, deployment); err != nil {
		return err
	}

	// Send success message
	sendMessage("Docker Compose deployment completed successfully", "success")

	return nil
}

func (s *ProjectService) DeployPiping(projectID uuid.UUID, pull bool) error {
	// Create a local channel to capture streaming output
	outputChan := make(chan docker.StreamMessage, 100)
	done := make(chan bool)

	// Start a goroutine to consume StreamMessages and display to terminal
	go func() {
		defer func() { done <- true }()
		for msg := range outputChan {
			// Display raw message content to terminal without modification
			fmt.Println(msg.Content)
		}
	}()

	// Use DeployStreaming internally (it now stores clean output in database)
	err := s.DeployStreaming(projectID, pull, outputChan)

	// Close channel and wait for goroutine to finish
	close(outputChan)
	<-done

	return err
}

// prepareDeployment handles the common setup logic for both streaming and piping deployments
func (s *ProjectService) prepareDeployment(
	projectID uuid.UUID,
	pull bool,
) (*domain.Project, string, domain.Deployment, *docker.ComposeProject, error) {
	// Get project
	project, err := s.Get(projectID)
	if err != nil {
		return nil, "", domain.Deployment{}, nil, fmt.Errorf("project not found: %w", err)
	}

	gitDir, err := project.GitDir()
	if err != nil {
		return nil, "", domain.Deployment{}, nil, fmt.Errorf("failed to get git directory: %w", err)
	}

	commitHash, err := s.gitService.GetLatestCommit(gitDir)
	if err != nil {
		slog.Error("Service operation failed",
			"layer", "service",
			"operation", "deploy_project",
			"project_id", project.ID,
			"error", err)
		return nil, "", domain.Deployment{}, nil, err
	}

	deployment := domain.NewDeployment(projectID, commitHash)
	deployment.Status = domain.DeploymentStatusStarted

	// Create deployment record immediately
	if err := s.deploymentRepository.Create(&deployment); err != nil {
		return nil, "", domain.Deployment{}, nil, fmt.Errorf("failed to create deployment record: %w", err)
	}

	// Log deployment start
	slog.Debug("Starting Docker Compose deployment",
		"project_id", project.ID,
		"project_name", project.Name,
		"deployment_id", deployment.ID,
		"commit_hash", commitHash,
		"compose_files", project.ComposeFiles,
		"pull", pull)

	composeProject := docker.NewComposeProject(project, s.config)

	return project, commitHash, deployment, composeProject, nil
}

// handleDeploymentError handles deployment errors consistently
func (s *ProjectService) handleDeploymentError(
	project *domain.Project,
	deployment *domain.Deployment,
	err error,
) error {
	// Update deployment record as failed and append error info to output
	deployment.Status = domain.DeploymentStatusFailed

	// Append error information to stderr
	if deployment.Stderr != "" {
		deployment.Stderr += "\n"
	}
	deployment.Stderr += fmt.Sprintf("ERROR: %v", err)

	// Update project status to error
	project.Status = domain.ProjectStatusError

	// Update both deployment and project records
	if updateErr := s.deploymentRepository.Update(deployment); updateErr != nil {
		slog.Error("Failed to update deployment record as failed",
			"deployment_id", deployment.ID,
			"project_id", deployment.ProjectID,
			"error", updateErr)
	}

	if updateErr := s.projectRepository.Update(project); updateErr != nil {
		slog.Error("Failed to update project status to error",
			"project_id", project.ID,
			"error", updateErr)
	}

	slog.Error(
		"Docker Compose up failed",
		"project_id", deployment.ProjectID,
		"deployment_id", deployment.ID,
		"error", err,
	)
	return fmt.Errorf("failed to start project: %w", err)
}

// completeDeployment handles the post-deployment database updates
func (s *ProjectService) completeDeployment(
	project *domain.Project,
	commitHash string,
	deployment domain.Deployment,
) error {
	slog.Debug(
		"Docker Compose project started",
		"project_id", project.ID,
	)

	// Update deployment
	deployment.Status = domain.DeploymentStatusCompleted

	// Update project
	project.Status = domain.ProjectStatusRunning
	project.LastCommit = &commitHash

	// TODO: Transaction
	if err := s.deploymentRepository.Update(&deployment); err != nil {
		return fmt.Errorf("failed to update deployment record: %w", err)
	}

	if err := s.projectRepository.Update(project); err != nil {
		return fmt.Errorf("failed to update project status: %w", err)
	}

	return nil
}

func (s *ProjectService) Stop(projectID uuid.UUID, removeVolumes bool) error {
	// Get project
	project, err := s.Get(projectID)
	if err != nil {
		return fmt.Errorf("project not found: %w", err)
	}

	// Stop Docker Compose
	slog.Info(
		"Stopping Docker Compose project",
		"project_id",
		project.ID,
		"project_name",
		project.Name,
	)

	composeProject := docker.NewComposeProject(project, s.config)

	stdout, stderr, err := composeProject.Down(removeVolumes)
	if err != nil {
		slog.Error(
			"Docker Compose down failed",
			"project_id",
			project.ID,
			"error",
			err,
			"stdout",
			stdout,
			"stderr",
			stderr,
		)
		return fmt.Errorf("failed to stop project: %w", err)
	}
	slog.Info(
		"Docker Compose project stopped",
		"project_id",
		project.ID,
		"stdout_length",
		len(stdout),
		"stderr_length",
		len(stderr),
	)

	project.Status = domain.ProjectStatusStopped
	return s.Update(project)
}

func (s *ProjectService) StopStreaming(projectID uuid.UUID, outputChan chan<- docker.StreamMessage) error {
	// Get project
	project, err := s.Get(projectID)
	if err != nil {
		return fmt.Errorf("project not found: %w", err)
	}

	// Stop Docker Compose
	slog.Info(
		"Stopping Docker Compose project with streaming",
		"project_id",
		project.ID,
		"project_name",
		project.Name,
	)

	composeProject := docker.NewComposeProject(project, s.config)

	// Helper function to send StreamMessage
	sendMessage := func(msg, msgType string) {
		outputChan <- docker.StreamMessage{Type: msgType, Content: msg}
	}

	sendMessage("Starting Docker Compose shutdown...", "info")

	// Create a capturing channel that forwards Docker stdout/stderr directly
	capturingChan := make(chan docker.StreamMessage, 100)
	done := make(chan bool)

	go func() {
		defer func() { done <- true }()
		for msg := range capturingChan {
			// Forward message directly to user
			outputChan <- msg
		}
	}()

	// Execute stop with streaming
	err = composeProject.DownStreaming(capturingChan)
	close(capturingChan) // Signal that we're done sending to the capturing channel
	<-done               // Wait for the goroutine to finish processing all messages

	if err != nil {
		slog.Error(
			"Docker Compose down failed",
			"project_id",
			project.ID,
			"error",
			err,
		)
		sendMessage(fmt.Sprintf("Stop failed: %v", err), "error")
		return fmt.Errorf("failed to stop project: %w", err)
	}

	sendMessage("Docker Compose project stopped successfully", "success")
	slog.Info(
		"Docker Compose project stopped",
		"project_id",
		project.ID,
	)

	project.Status = domain.ProjectStatusStopped
	err = s.Update(project)
	if err != nil {
		return fmt.Errorf("failed to update project status: %w", err)
	}

	// Send unified message with both display text and project state
	sendMessage("Docker Compose shutdown completed successfully", "success")

	return nil
}

func (s *ProjectService) StopPiping(projectID uuid.UUID) error {
	// Get project
	project, err := s.Get(projectID)
	if err != nil {
		return fmt.Errorf("project not found: %w", err)
	}

	// Stop Docker Compose
	slog.Debug(
		"Stopping Docker Compose project with piping",
		"project_id",
		project.ID,
		"project_name",
		project.Name,
	)

	composeProject := docker.NewComposeProject(project, s.config)

	err = composeProject.DownPiping()
	if err != nil {
		slog.Error(
			"Docker Compose down failed",
			"project_id",
			project.ID,
			"error",
			err,
		)
		return fmt.Errorf("failed to stop project: %w", err)
	}
	slog.Debug(
		"Docker Compose project stopped",
		"project_id",
		project.ID,
	)

	project.Status = domain.ProjectStatusStopped
	return s.Update(project)
}

func (s *ProjectService) Remove(projectID uuid.UUID, removeVolumes bool) error {
	// Get project
	project, err := s.Get(projectID)
	if err != nil {
		return fmt.Errorf("project not found: %w", err)
	}

	// Stop Docker Compose project if running
	if err := s.Stop(projectID, removeVolumes); err != nil {
		slog.Warn("Failed to stop project before removal", "project_id", project.ID, "error", err)
		return fmt.Errorf("failed to stop project before removal: %w", err)
	}

	// Rename project directory to indicate deletion instead of removing it
	// This avoids issues with root-owned files from Docker containers
	deletedDirPath := domain.GetDeletedDirectoryPath(project.WorkingDir)
	if err := os.Rename(project.WorkingDir, deletedDirPath); err != nil {
		slog.Warn("Failed to rename project directory, continuing with deletion",
			"project_id", project.ID,
			"from", project.WorkingDir,
			"to", deletedDirPath,
			"error", err)
	} else {
		slog.Info("Project directory renamed to indicate deletion",
			"project_id", project.ID,
			"from", project.WorkingDir,
			"to", deletedDirPath)
	}

	// Delete project from database
	if err := s.projectRepository.Delete(projectID); err != nil {
		return fmt.Errorf("failed to delete project from database: %w", err)
	}

	slog.Info(
		"Project removed successfully",
		"project_id",
		project.ID,
		"working_dir",
		project.WorkingDir,
	)
	return nil
}

func (s *ProjectService) GetLogs(projectID uuid.UUID) (string, string, error) {
	// Get project
	project, err := s.Get(projectID)
	if err != nil {
		return "", "", fmt.Errorf("project not found: %w", err)
	}

	// Get logs using Docker Compose
	slog.Info(
		"Getting logs for Docker Compose project",
		"project_id",
		project.ID,
		"project_name",
		project.Name,
	)

	composeProject := docker.NewComposeProject(project, s.config)
	stdout, stderr, err := composeProject.Logs()
	if err != nil {
		slog.Error(
			"Failed to get logs",
			"project_id",
			project.ID,
			"error",
			err,
		)
		return "", "", fmt.Errorf("failed to get logs: %w", err)
	}

	slog.Info(
		"Logs retrieved successfully",
		"project_id",
		project.ID,
		"project_name",
		project.Name,
		"stdout_length",
		len(stdout),
		"stderr_length",
		len(stderr),
	)
	return stdout, stderr, nil
}

func (s *ProjectService) GetLogsPiping(projectID uuid.UUID) error {
	// Get project
	project, err := s.Get(projectID)
	if err != nil {
		return fmt.Errorf("project not found: %w", err)
	}

	// Stream logs using Docker Compose with direct piping
	slog.Debug(
		"Streaming logs for Docker Compose project with piping",
		"project_id",
		project.ID,
		"project_name",
		project.Name,
	)

	composeProject := docker.NewComposeProject(project, s.config)

	err = composeProject.LogsPiping()
	if err != nil {
		slog.Error(
			"Failed to stream logs",
			"project_id",
			project.ID,
			"error",
			err,
		)
		return fmt.Errorf("failed to stream logs: %w", err)
	}
	slog.Debug(
		"Logs streaming completed",
		"project_id",
		project.ID,
		"project_name",
		project.Name,
	)
	return nil
}

func (s *ProjectService) GetConfig(projectID uuid.UUID) (string, string, error) {
	// Get project
	project, err := s.Get(projectID)
	if err != nil {
		return "", "", fmt.Errorf("project not found: %w", err)
	}

	// Get configuration using Docker Compose
	slog.Debug(
		"Getting Docker Compose configuration",
		"project_id",
		project.ID,
		"project_name",
		project.Name,
	)

	composeProject := docker.NewComposeProject(project, s.config)

	stdout, stderr, err := composeProject.GetConfig()
	if err != nil {
		slog.Error(
			"Failed to get configuration",
			"project_id",
			project.ID,
			"error",
			err,
		)
		return "", "", fmt.Errorf("failed to get configuration: %w", err)
	}

	slog.Debug(
		"Configuration retrieved successfully",
		"project_id",
		project.ID,
		"project_name",
		project.Name,
		"stdout_length",
		len(stdout),
		"stderr_length",
		len(stderr),
	)
	return stdout, stderr, nil
}

// GetStatus gets the current status of a project's containers
func (s *ProjectService) GetStatus(projectID uuid.UUID) (*docker.ComposeStatus, error) {
	// Get project
	project, err := s.Get(projectID)
	if err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}

	// Get status using Docker Compose
	slog.Debug(
		"Getting Docker Compose status",
		"project_id",
		project.ID,
		"project_name",
		project.Name,
	)

	composeProject := docker.NewComposeProject(project, s.config)

	status, err := composeProject.Status()
	if err != nil {
		slog.Error(
			"Failed to get status",
			"project_id",
			project.ID,
			"error",
			err,
		)
		return nil, fmt.Errorf("failed to get status: %w", err)
	}
	slog.Debug(
		"Status retrieved successfully",
		"project_id",
		project.ID,
		"project_name",
		project.Name,
		"status",
		status.Status,
	)
	return status, nil
}

func (s *ProjectService) pullLatestChanges(project *domain.Project) error {
	slog.Debug("Pulling latest changes", "project_id", project.ID, "git_url", project.GitURL)

	gitDir, err := project.GitDir()
	if err != nil {
		return fmt.Errorf("failed to get git directory: %w", err)
	}

	if err = s.gitService.Pull(project.GitBranch, project.GitAuth, gitDir); err != nil {
		slog.Error("Failed to pull changes", "project_id", project.ID, "error", err)
		return fmt.Errorf("failed to pull changes: %w", err)
	}

	slog.Debug("Git pull completed", "project_id", project.ID)
	return nil
}

// ListDeployments lists all deployments for a specific project
func (s *ProjectService) ListDeployments(projectID uuid.UUID) ([]*domain.Deployment, error) {
	// First verify the project exists
	_, err := s.Get(projectID)
	if err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}

	deployments, err := s.deploymentRepository.ListByProjectID(projectID)
	if err != nil {
		slog.Error("Service operation failed",
			"layer", "service",
			"operation", "list_deployments",
			"project_id", projectID,
			"error", err)
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}

	slog.Debug("Listed deployments for project",
		"project_id", projectID,
		"deployment_count", len(deployments))

	return deployments, nil
}

// NewProjectService creates a new ProjectService with dependency injection
func NewProjectService(
	projectRepository repository.ProjectRepository,
	deploymentRepository repository.DeploymentRepository,
	gitService *git.GitService,
	cfg *config.Config,
) *ProjectService {
	return &ProjectService{
		projectRepository:    projectRepository,
		deploymentRepository: deploymentRepository,
		gitService:           gitService,
		config:               cfg,
	}
}
