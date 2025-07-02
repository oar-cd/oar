package services

import (
	"fmt"
	"log/slog"
	"os/exec"
	"path/filepath"
)

// DockerComposeProjectService is a placeholder for Docker Compose related operations.
type DockerComposeProjectService struct{}

// Ensure DockerComposeProjectService implements DockerComposeExecutor
var _ DockerComposeExecutor = (*DockerComposeProjectService)(nil)

func (d *DockerComposeProjectService) Up(project *Project) (*DeploymentResult, error) {
	gitDir, err := project.GitDir()
	if err != nil {
		slog.Error("Failed to get Git directory for project",
			"project_name", project.Name,
			"error", err)
		return nil, fmt.Errorf("failed to get git directory: %w", err)
	}

	// Build docker compose command
	args := []string{
		"compose",
		"--project-name", project.Name,
	}

	// Add compose files to the command
	for _, file := range project.ComposeFiles {
		args = append(args, "--file", filepath.Join(gitDir, file))
	}

	// Argument and common flags
	args = append(args,
		"up",
		"--detach", "--wait", "--quiet-pull", "--no-color", "--remove-orphans",
	)

	slog.Debug("Executing Docker Compose command",
		"command", "docker",
		"args", args,
		"git_dir", gitDir)

	// Execute command
	cmd := exec.Command("docker", args...)
	cmd.Dir = gitDir
	cmd.Env = append(cmd.Env, "COMPOSE_PROJECT_NAME="+project.Name) // TODO: Add stuff from EnvironmentFiles

	result := &DeploymentResult{
		CommandLine: cmd.String(),
		Status:      DeploymentStatusStarted,
	}

	// Capture output
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil {
		slog.Error("Docker Compose command failed",
			"command", cmd.String(),
			"error", err,
			"output", outputStr)
		result.Status = DeploymentStatusFailed
		return result, fmt.Errorf("docker compose command failed: %w", err)
	}

	slog.Debug("Docker Compose command completed successfully",
		"project_name", project.Name,
		"output_length", len(outputStr))

	return result, nil
}

func (d *DockerComposeProjectService) Down(project *Project) (string, error) {
	gitDir, err := project.GitDir()
	if err != nil {
		slog.Error("Failed to get Git directory for project",
			"project_name", project.Name,
			"error", err)
		return "", fmt.Errorf("failed to get git directory: %w", err)
	}

	// Build docker compose command
	args := []string{
		"compose",
		"--project-name", project.Name,
	}

	// Add compose files to the command
	for _, file := range project.ComposeFiles {
		args = append(args, "--file", filepath.Join(gitDir, file))
	}
	// Argument and common flags
	args = append(args,
		"down",
		"--remove-orphans",
	)

	slog.Debug("Executing Docker Compose down command",
		"command", "docker",
		"args", args,
		"git_dir", gitDir)

	// Execute command
	cmd := exec.Command("docker", args...)
	cmd.Dir = gitDir

	// Capture output
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil {
		slog.Error("Docker Compose down command failed",
			"command", "docker",
			"args", args,
			"error", err,
			"output", outputStr)
		return outputStr, fmt.Errorf("docker compose down command failed: %w", err)
	}

	slog.Debug("Docker Compose down command completed successfully",
		"project_name", project.Name,
		"output_length", len(outputStr))
	return outputStr, nil
}

// NewDockerComposeService creates a new instance of DockerComposeService.
func NewDockerComposeService() *DockerComposeProjectService {
	return &DockerComposeProjectService{}
}
