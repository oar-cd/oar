package services

import (
	"bufio"
	"fmt"
	"log/slog"
	"os/exec"
	"path/filepath"
)

type ComposeProject struct {
	// Name is the name of the Docker Compose project.
	Name string
	// WorkingDir is the directory where the Docker Compose files are located.
	WorkingDir string
	// ComposeFiles is a list of Docker Compose files for the project.
	ComposeFiles []string
	// EnvironmentFiles is a list of environment files for the project.
	EnvironmentFiles []string
}

// Ensure ComposeProject implements ComposeProjectInterface
var _ ComposeProjectInterface = (*ComposeProject)(nil)

func NewComposeProject(p *Project) *ComposeProject {
	gitDir, err := p.GitDir()
	if err != nil {
		slog.Error("Failed to get Git directory for project",
			"project_name", p.Name,
			"error", err)
		return nil
	}

	return &ComposeProject{
		Name:             p.Name,
		WorkingDir:       gitDir,
		ComposeFiles:     p.ComposeFiles,
		EnvironmentFiles: p.EnvironmentFiles,
	}
}

func (p *ComposeProject) Up() (string, error) {
	cmd, err := p.commandUp()
	if err != nil {
		slog.Error("Failed to prepare Docker Compose up command",
			"project_name", p.Name,
			"error", err)
		return "", fmt.Errorf("failed to prepare docker compose up command: %w", err)
	}

	return p.executeCommand(cmd)
}

func (p *ComposeProject) UpStreaming(outputChan chan<- string) error {
	cmd, err := p.commandUp()
	if err != nil {
		slog.Error("Failed to prepare Docker Compose up command",
			"project_name", p.Name,
			"error", err)
		return fmt.Errorf("failed to prepare docker compose up command: %w", err)
	}
	return p.executeCommandStreaming(cmd, outputChan)
}

func (p *ComposeProject) Down() (string, error) {
	cmd, err := p.commandDown()
	if err != nil {
		slog.Error("Failed to prepare Docker Compose down command",
			"project_name", p.Name,
			"error", err)
		return "", fmt.Errorf("failed to prepare docker compose down command: %w", err)
	}

	return p.executeCommand(cmd)
}

func (p *ComposeProject) DownStreaming(outputChan chan<- string) error {
	cmd, err := p.commandDown()
	if err != nil {
		slog.Error("Failed to prepare Docker Compose down command",
			"project_name", p.Name,
			"error", err)
		return fmt.Errorf("failed to prepare docker compose down command: %w", err)
	}

	return p.executeCommandStreaming(cmd, outputChan)
}

func (p *ComposeProject) Logs() (string, error) {
	cmd, err := p.commandLogs()
	if err != nil {
		slog.Error("Failed to prepare Docker Compose up command",
			"project_name", p.Name,
			"error", err)
		return "", fmt.Errorf("failed to prepare docker compose up command: %w", err)
	}

	return p.executeCommand(cmd)
}

func (p *ComposeProject) LogsStreaming(outputChan chan<- string) error {
	cmd, err := p.commandLogs()
	if err != nil {
		slog.Error("Failed to prepare Docker Compose logs command",
			"project_name", p.Name,
			"error", err)
		return fmt.Errorf("failed to prepare docker compose logs command: %w", err)
	}

	return p.executeCommandStreaming(cmd, outputChan)
}

func (p *ComposeProject) prepareCommand(command string, args []string) (*exec.Cmd, error) {
	// Build docker compose command
	commandArgs := []string{
		"compose",
		"--project-name", p.Name,
	}

	// Add compose files to the command
	for _, file := range p.ComposeFiles {
		commandArgs = append(commandArgs, "--file", filepath.Join(p.WorkingDir, file))
	}

	// Add the specific command and its arguments
	commandArgs = append(commandArgs, command)
	commandArgs = append(commandArgs, args...)

	slog.Debug("Executing Docker Compose command",
		"command", "docker",
		"args", commandArgs,
		"working_dir", p.WorkingDir)

	// Create command
	cmd := exec.Command("docker", commandArgs...)
	cmd.Dir = p.WorkingDir

	return cmd, nil
}

func (p *ComposeProject) executeCommand(cmd *exec.Cmd) (string, error) {
	out, err := cmd.CombinedOutput()
	output := string(out)
	if err != nil {
		slog.Error("Docker Compose up command failed",
			"project_name", p.Name,
			"error", err,
			"output", output)
		return "", fmt.Errorf("docker compose up command failed: %w", err)
	}
	return output, nil
}

func (p *ComposeProject) executeCommandStreaming(cmd *exec.Cmd, outputChan chan<- string) error {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		slog.Error("Failed to get stdout pipe for Docker Compose command",
			"command", cmd.String(),
			"error", err)
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		slog.Error("Failed to get stderr pipe for Docker Compose command",
			"command", cmd.String(),
			"error", err)
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	// Start the command
	err = cmd.Start()
	if err != nil {
		slog.Error("Failed to start Docker Compose command",
			"command", cmd.String(),
			"error", err)
		return fmt.Errorf("failed to start docker compose command: %w", err)
	}

	// Stream stdout
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			outputChan <- scanner.Text()
		}
	}()

	// Stream stderr
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			outputChan <- scanner.Text()
		}
	}()

	err = cmd.Wait()
	if err != nil {
		slog.Error("Docker Compose command failed",
			"command", cmd.String(),
			"error", err)
		return fmt.Errorf("docker compose command failed: %w", err)
	}

	slog.Debug("Docker Compose command completed successfully")

	return nil
}

func (p *ComposeProject) commandUp() (*exec.Cmd, error) {
	cmd, err := p.prepareCommand("up", []string{"--detach", "--wait", "--quiet-pull", "--no-color", "--remove-orphans"})
	if err != nil {
		slog.Error("Failed to prepare Docker Compose up command",
			"project_name", p.Name,
			"error", err)
		return nil, fmt.Errorf("failed to prepare docker compose up command: %w", err)
	}

	return cmd, nil
}

func (p *ComposeProject) commandDown() (*exec.Cmd, error) {
	cmd, err := p.prepareCommand("down", []string{"--remove-orphans"})
	if err != nil {
		slog.Error("Failed to prepare Docker Compose down command",
			"project_name", p.Name,
			"error", err)
		return nil, fmt.Errorf("failed to prepare docker compose down command: %w", err)
	}

	return cmd, nil
}

func (p *ComposeProject) commandLogs() (*exec.Cmd, error) {
	cmd, err := p.prepareCommand("logs", []string{"--follow"})
	if err != nil {
		slog.Error("Failed to prepare Docker Compose logs command",
			"project_name", p.Name,
			"error", err)
		return nil, fmt.Errorf("failed to prepare docker compose logs command: %w", err)
	}

	return cmd, nil
}
