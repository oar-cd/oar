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

func (d *DockerComposeProjectService) Up(projectConfig *ProjectConfig, deploymentConfig *DeploymentConfig) (string, error) {
	// Build docker compose command
	args := []string{
		"compose",
		"--project-name", projectConfig.Name,
	}

	// Add compose files to the command
	for _, file := range projectConfig.ComposeFiles {
		args = append(args, "--file", filepath.Join(projectConfig.WorkingDir, file))
	}

	// Argument and common flags
	args = append(args,
		"up",
		"--quiet-pull", "--no-color", "--remove-orphans",
	)

	slog.Debug("Executing Docker Compose command",
		"command", "docker",
		"args", args,
		"working_dir", projectConfig.WorkingDir)

	// Execute command
	cmd := exec.Command("docker", args...)
	cmd.Dir = projectConfig.WorkingDir
	cmd.Env = append(cmd.Env, "COMPOSE_PROJECT_NAME="+projectConfig.Name) // TODO: Add stuff from EnvironmentFiles

	// Capture output
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil {
		slog.Error("Docker Compose command failed",
			"command", "docker",
			"args", args,
			"error", err,
			"output", outputStr)
		return outputStr, fmt.Errorf("docker compose command failed: %w", err)
	}

	slog.Debug("Docker Compose command completed successfully",
		"project_name", projectConfig.Name,
		"output_length", len(outputStr))
	return outputStr, nil
}

func (d *DockerComposeProjectService) Down(projectConfig *ProjectConfig) (string, error) {
	// Build docker compose command
	args := []string{
		"compose",
		"--project-name", projectConfig.Name,
	}

	// Add compose files to the command
	for _, file := range projectConfig.ComposeFiles {
		args = append(args, "--file", filepath.Join(projectConfig.WorkingDir, file))
	}
	// Argument and common flags
	args = append(args,
		"down",
		"--remove-orphans",
	)

	slog.Debug("Executing Docker Compose down command",
		"command", "docker",
		"args", args,
		"working_dir", projectConfig.WorkingDir)

	// Execute command
	cmd := exec.Command("docker", args...)
	cmd.Dir = projectConfig.WorkingDir

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
		"project_name", projectConfig.Name,
		"output_length", len(outputStr))
	return outputStr, nil
}

// NewDockerComposeService creates a new instance of DockerComposeService.
func NewDockerComposeService() *DockerComposeProjectService {
	return &DockerComposeProjectService{}
}
