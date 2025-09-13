// Package services provides interfaces and implementations for various services in Oar.
package services

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/gosimple/slug"
)

type ProjectStatus int

const (
	ProjectStatusRunning ProjectStatus = iota
	ProjectStatusStopped
	ProjectStatusError
	ProjectStatusUnknown
)

func (s ProjectStatus) String() string {
	switch s {
	case ProjectStatusRunning:
		return "running"
	case ProjectStatusStopped:
		return "stopped"
	case ProjectStatusError:
		return "error"
	case ProjectStatusUnknown:
		return "unknown"
	default:
		return "unknown"
	}
}

func ParseProjectStatus(s string) (ProjectStatus, error) {
	switch s {
	case "running":
		return ProjectStatusRunning, nil
	case "stopped":
		return ProjectStatusStopped, nil
	case "error":
		return ProjectStatusError, nil
	case "unknown":
		return ProjectStatusUnknown, nil
	default:
		return ProjectStatusUnknown, fmt.Errorf("invalid project status: %q", s)
	}
}

// GetDeletedDirectoryPath calculates the path where a project directory will be moved when deleted
func GetDeletedDirectoryPath(workingDir string) string {
	deletedDirName := fmt.Sprintf("deleted-%s", filepath.Base(workingDir))
	return filepath.Join(filepath.Dir(workingDir), deletedDirName)
}

// ProjectService provides methods to manage Docker Compose projects.
type ProjectService struct {
	projectRepository    ProjectRepository
	deploymentRepository DeploymentRepository
	gitService           GitExecutor
	config               *Config
}

// Ensure ProjectService implements ProjectManager
var _ ProjectManager = (*ProjectService)(nil)

// List returns all projects
func (s *ProjectService) List() ([]*Project, error) {
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
func (s *ProjectService) Get(id uuid.UUID) (*Project, error) {
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
func (s *ProjectService) Create(project *Project) (*Project, error) {
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

func (s *ProjectService) Update(project *Project) error {
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
	outputChan chan<- string,
) error {
	project, commitHash, deployment, composeProject, err := s.prepareDeployment(projectID, pull)
	if err != nil {
		return err
	}

	// Create a buffer to capture clean output for the deployment record
	var outputBuffer strings.Builder

	// Helper function to capture clean message and send JSON to channel
	captureAndSendJSON := func(cleanMsg, msgType, source string) {
		// Store clean message for database
		outputBuffer.WriteString(cleanMsg + "\n")

		// Create JSON message for web UI
		msgData := map[string]string{
			"type":    msgType,
			"message": cleanMsg,
		}
		if source != "" {
			msgData["source"] = source
		}

		if jsonMsg, err := json.Marshal(msgData); err == nil {
			outputChan <- string(jsonMsg)
		}
	}

	// Streaming-specific messages
	if pull {
		captureAndSendJSON("Pulling latest changes from Git...", "info", "oar")

		// Get commit hash before pull
		gitDir, err := project.GitDir()
		if err != nil {
			errMsg := fmt.Sprintf("Failed to get git directory: %v", err)
			captureAndSendJSON(errMsg, "error", "oar")
			return err
		}

		beforeCommit, err := s.gitService.GetLatestCommit(gitDir)
		if err != nil {
			slog.Warn("Failed to get commit hash before pull", "project_id", project.ID, "error", err)
			beforeCommit = "unknown"
		}

		if err := s.pullLatestChanges(project); err != nil {
			errMsg := fmt.Sprintf("Failed to pull latest changes: %v", err)
			captureAndSendJSON(errMsg, "error", "oar")
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
		captureAndSendJSON(successMsg, "success", "oar")
	}

	captureAndSendJSON("Starting Docker Compose deployment...", "info", "oar")

	// Create a capturing channel that forwards to the original channel
	capturingChan := make(chan string, 100) // buffered channel
	done := make(chan bool)

	go func() {
		defer func() { done <- true }()
		for msg := range capturingChan {
			// msg is now clean output from Docker, store it and wrap in JSON
			outputBuffer.WriteString(msg + "\n")

			// Wrap in JSON for web UI
			msgData := map[string]string{
				"type":    "docker",
				"message": msg,
			}
			if jsonMsg, err := json.Marshal(msgData); err == nil {
				outputChan <- string(jsonMsg)
			}
		}
	}()

	// Execute deployment with streaming
	err = composeProject.UpStreaming(capturingChan)
	close(capturingChan) // Signal that we're done sending to the capturing channel
	<-done               // Wait for the goroutine to finish processing all messages

	// Store the captured output in the deployment record
	deployment.Output = outputBuffer.String()

	if err != nil {
		return s.handleDeploymentError(&deployment, err)
	}

	// Complete deployment
	if err := s.completeDeployment(project, commitHash, deployment); err != nil {
		return err
	}

	// Send success message
	captureAndSendJSON("Docker Compose deployment completed successfully", "success", "oar")

	return nil
}

func (s *ProjectService) DeployPiping(projectID uuid.UUID, pull bool) error {
	// Create a local channel to capture streaming output
	outputChan := make(chan string, 100)
	done := make(chan bool)

	// Start a goroutine to consume JSON messages and display clean messages to terminal
	go func() {
		defer func() { done <- true }()
		for msg := range outputChan {
			// Extract clean message from JSON
			var msgData map[string]string
			if err := json.Unmarshal([]byte(msg), &msgData); err == nil {
				// Display clean message to terminal
				fmt.Println(msgData["message"])
			}
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
) (*Project, string, Deployment, *ComposeProject, error) {
	// Get project
	project, err := s.Get(projectID)
	if err != nil {
		return nil, "", Deployment{}, nil, fmt.Errorf("project not found: %w", err)
	}

	gitDir, err := project.GitDir()
	if err != nil {
		return nil, "", Deployment{}, nil, fmt.Errorf("failed to get git directory: %w", err)
	}

	commitHash, err := s.gitService.GetLatestCommit(gitDir)
	if err != nil {
		slog.Error("Service operation failed",
			"layer", "service",
			"operation", "deploy_project",
			"project_id", project.ID,
			"error", err)
		return nil, "", Deployment{}, nil, err
	}

	deployment := NewDeployment(projectID, commitHash)
	deployment.Status = DeploymentStatusStarted

	// Create deployment record immediately
	if err := s.deploymentRepository.Create(&deployment); err != nil {
		return nil, "", Deployment{}, nil, fmt.Errorf("failed to create deployment record: %w", err)
	}

	// Log deployment start
	slog.Debug("Starting Docker Compose deployment",
		"project_id", project.ID,
		"project_name", project.Name,
		"deployment_id", deployment.ID,
		"commit_hash", commitHash,
		"compose_files", project.ComposeFiles,
		"pull", pull)

	composeProject := NewComposeProject(project, s.config)

	return project, commitHash, deployment, composeProject, nil
}

// handleDeploymentError handles deployment errors consistently
func (s *ProjectService) handleDeploymentError(deployment *Deployment, err error) error {
	// Update deployment record as failed and append error info to output
	deployment.Status = DeploymentStatusFailed

	// Append error information to the existing output
	if deployment.Output != "" {
		deployment.Output += "\n"
	}
	deployment.Output += fmt.Sprintf("ERROR: %v", err)

	if updateErr := s.deploymentRepository.Update(deployment); updateErr != nil {
		slog.Error("Failed to update deployment record as failed",
			"deployment_id", deployment.ID,
			"project_id", deployment.ProjectID,
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
func (s *ProjectService) completeDeployment(project *Project, commitHash string, deployment Deployment) error {
	slog.Debug(
		"Docker Compose project started",
		"project_id", project.ID,
	)

	// Update deployment
	deployment.Status = DeploymentStatusCompleted

	// Update project
	project.Status = ProjectStatusRunning
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

func (s *ProjectService) Stop(projectID uuid.UUID) error {
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

	composeProject := NewComposeProject(project, s.config)

	output, err := composeProject.Down()
	if err != nil {
		slog.Error(
			"Docker Compose down failed",
			"project_id",
			project.ID,
			"error",
			err,
			"output",
			output,
		)
		return fmt.Errorf("failed to stop project: %w", err)
	}
	slog.Info(
		"Docker Compose project stopped",
		"project_id",
		project.ID,
		"output_length",
		len(output),
	)

	project.Status = ProjectStatusStopped
	return s.Update(project)
}

func (s *ProjectService) StopStreaming(projectID uuid.UUID, outputChan chan<- string) error {
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

	composeProject := NewComposeProject(project, s.config)

	// Helper function to send JSON-wrapped messages
	sendJSON := func(msgType, message, source string) {
		msgData := map[string]string{
			"type":    msgType,
			"message": message,
		}
		if source != "" {
			msgData["source"] = source
		}

		if jsonMsg, err := json.Marshal(msgData); err == nil {
			outputChan <- string(jsonMsg)
		}
	}

	sendJSON("info", "Starting Docker Compose shutdown...", "oar")

	// Create a capturing channel that forwards to the original channel
	capturingChan := make(chan string, 100) // buffered channel
	done := make(chan bool)

	go func() {
		defer func() { done <- true }()
		for msg := range capturingChan {
			// Wrap Docker output in JSON for web UI
			msgData := map[string]string{
				"type":    "docker",
				"message": msg,
			}
			if jsonMsg, err := json.Marshal(msgData); err == nil {
				outputChan <- string(jsonMsg)
			}
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
		return fmt.Errorf("failed to stop project: %w", err)
	}
	slog.Info(
		"Docker Compose project stopped",
		"project_id",
		project.ID,
	)

	project.Status = ProjectStatusStopped
	err = s.Update(project)
	if err != nil {
		return fmt.Errorf("failed to update project status: %w", err)
	}

	// Send unified message with both display text and project state
	sendJSON("success", "Docker Compose shutdown completed successfully", "")

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

	composeProject := NewComposeProject(project, s.config)

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

	project.Status = ProjectStatusStopped
	return s.Update(project)
}

func (s *ProjectService) Remove(projectID uuid.UUID) error {
	// Get project
	project, err := s.Get(projectID)
	if err != nil {
		return fmt.Errorf("project not found: %w", err)
	}

	// Stop Docker Compose project if running
	if err := s.Stop(projectID); err != nil {
		slog.Warn("Failed to stop project before removal", "project_id", project.ID, "error", err)
		return fmt.Errorf("failed to stop project before removal: %w", err)
	}

	// Rename project directory to indicate deletion instead of removing it
	// This avoids issues with root-owned files from Docker containers
	deletedDirPath := GetDeletedDirectoryPath(project.WorkingDir)
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

func (s *ProjectService) GetLogsStreaming(projectID uuid.UUID, outputChan chan<- string) error {
	// Get projectID
	project, err := s.Get(projectID)
	if err != nil {
		return fmt.Errorf("project not found: %w", err)
	}

	// Stream logs using Docker Compose
	slog.Info(
		"Streaming logs for Docker Compose project",
		"project_id",
		project.ID,
		"project_name",
		project.Name,
	)

	composeProject := NewComposeProject(project, s.config)

	// Create a capturing channel that forwards to the original channel
	capturingChan := make(chan string, 100) // buffered channel
	done := make(chan bool)

	go func() {
		defer func() { done <- true }()
		for msg := range capturingChan {
			// Wrap Docker log output in JSON for web UI
			msgData := map[string]string{
				"type":    "docker",
				"message": msg,
			}
			if jsonMsg, err := json.Marshal(msgData); err == nil {
				outputChan <- string(jsonMsg)
			}
		}
	}()

	// Execute logs with streaming
	err = composeProject.LogsStreaming(capturingChan)
	close(capturingChan) // Signal that we're done sending to the capturing channel
	<-done               // Wait for the goroutine to finish processing all messages

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
	slog.Info(
		"Logs streaming completed",
		"project_id",
		project.ID,
		"project_name",
		project.Name,
	)
	return nil
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

	composeProject := NewComposeProject(project, s.config)

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

func (s *ProjectService) GetConfig(projectID uuid.UUID) (string, error) {
	// Get project
	project, err := s.Get(projectID)
	if err != nil {
		return "", fmt.Errorf("project not found: %w", err)
	}

	// Get configuration using Docker Compose
	slog.Debug(
		"Getting Docker Compose configuration",
		"project_id",
		project.ID,
		"project_name",
		project.Name,
	)

	composeProject := NewComposeProject(project, s.config)

	output, err := composeProject.GetConfig()
	if err != nil {
		slog.Error(
			"Failed to get configuration",
			"project_id",
			project.ID,
			"error",
			err,
		)
		return "", fmt.Errorf("failed to get configuration: %w", err)
	}
	slog.Debug(
		"Configuration retrieved successfully",
		"project_id",
		project.ID,
		"project_name",
		project.Name,
		"output_length",
		len(output),
	)
	return output, nil
}

// GetStatus gets the current status of a project's containers
func (s *ProjectService) GetStatus(projectID uuid.UUID) (*ComposeStatus, error) {
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

	composeProject := NewComposeProject(project, s.config)

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

func (s *ProjectService) pullLatestChanges(project *Project) error {
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
func (s *ProjectService) ListDeployments(projectID uuid.UUID) ([]*Deployment, error) {
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
	projectRepository ProjectRepository,
	deploymentRepository DeploymentRepository,
	gitService GitExecutor,
	config *Config,
) *ProjectService {
	return &ProjectService{
		projectRepository:    projectRepository,
		deploymentRepository: deploymentRepository,
		gitService:           gitService,
		config:               config,
	}
}
