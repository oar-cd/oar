package services

import (
	"bufio"
	"encoding/json"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

type ContainerInfo struct {
	Service    string `json:"Service"`
	Name       string `json:"Name"`
	State      string `json:"State"`
	Status     string `json:"Status"`
	RunningFor string `json:"RunningFor"`
}

type ComposeStatus struct {
	Status     string
	Containers []ContainerInfo
	Uptime     string
}

type ComposeProject struct {
	// Name is the name of the Docker Compose project.
	Name string
	// WorkingDir is the directory where the Docker Compose files are located.
	WorkingDir string
	// ComposeFiles is a list of Docker Compose files for the project.
	ComposeFiles []string
	// Variables contains variables in KEY=value format
	Variables []string
	// Config holds configuration for docker commands and timeouts
	Config *Config
}

// Ensure ComposeProject implements ComposeProjectInterface
var _ ComposeProjectInterface = (*ComposeProject)(nil)

func NewComposeProject(p *Project, config *Config) *ComposeProject {
	gitDir, err := p.GitDir()
	if err != nil {
		slog.Error("Service operation failed",
			"layer", "docker_compose",
			"operation", "create_compose_project",
			"project_name", p.Name,
			"error", err)
		return nil
	}

	return &ComposeProject{
		Name:         p.Name,
		WorkingDir:   gitDir,
		ComposeFiles: p.ComposeFiles,
		Variables:    p.Variables,
		Config:       config,
	}
}

func (p *ComposeProject) Up() (string, error) {
	cmd, err := p.commandUp()
	if err != nil {
		slog.Error("Service operation failed",
			"layer", "docker_compose",
			"operation", "docker_compose_up",
			"project_name", p.Name,
			"error", err)
		return "", err
	}

	return p.executeCommand(cmd)
}

func (p *ComposeProject) UpStreaming(outputChan chan<- string) error {
	cmd, err := p.commandUp()
	if err != nil {
		slog.Error("Service operation failed",
			"layer", "docker_compose",
			"operation", "docker_compose_up",
			"project_name", p.Name,
			"error", err)
		return err
	}
	return p.executeCommandStreaming(cmd, outputChan)
}

func (p *ComposeProject) UpPiping() error {
	cmd, err := p.commandUp()
	if err != nil {
		slog.Error("Service operation failed",
			"layer", "docker_compose",
			"operation", "docker_compose_up",
			"project_name", p.Name,
			"error", err)
		return err
	}
	return p.executeCommandPiping(cmd)
}

func (p *ComposeProject) Down() (string, error) {
	cmd, err := p.commandDown()
	if err != nil {
		slog.Error("Service operation failed",
			"layer", "docker_compose",
			"operation", "docker_compose_down",
			"project_name", p.Name,
			"error", err)
		return "", err
	}

	return p.executeCommand(cmd)
}

func (p *ComposeProject) DownStreaming(outputChan chan<- string) error {
	cmd, err := p.commandDown()
	if err != nil {
		slog.Error("Service operation failed",
			"layer", "docker_compose",
			"operation", "docker_compose_down",
			"project_name", p.Name,
			"error", err)
		return err
	}

	return p.executeCommandStreaming(cmd, outputChan)
}

func (p *ComposeProject) DownPiping() error {
	cmd, err := p.commandDown()
	if err != nil {
		slog.Error("Service operation failed",
			"layer", "docker_compose",
			"operation", "docker_compose_down",
			"project_name", p.Name,
			"error", err)
		return err
	}
	return p.executeCommandPiping(cmd)
}

func (p *ComposeProject) Logs() (string, error) {
	cmd, err := p.commandLogs()
	if err != nil {
		slog.Error("Service operation failed",
			"layer", "docker_compose",
			"operation", "docker_compose_logs",
			"project_name", p.Name,
			"error", err)
		return "", err
	}

	return p.executeCommand(cmd)
}

func (p *ComposeProject) LogsStreaming(outputChan chan<- string) error {
	cmd, err := p.commandLogs()
	if err != nil {
		slog.Error("Service operation failed",
			"layer", "docker_compose",
			"operation", "docker_compose_logs",
			"project_name", p.Name,
			"error", err)
		return err
	}

	return p.executeCommandStreaming(cmd, outputChan)
}

func (p *ComposeProject) LogsPiping() error {
	cmd, err := p.commandLogs()
	if err != nil {
		slog.Error("Service operation failed",
			"layer", "docker_compose",
			"operation", "docker_compose_logs",
			"project_name", p.Name,
			"error", err)
		return err
	}
	return p.executeCommandPiping(cmd)
}

func (p *ComposeProject) GetConfig() (string, error) {
	cmd, err := p.commandConfig()
	if err != nil {
		slog.Error("Service operation failed",
			"layer", "docker_compose",
			"operation", "docker_compose_config",
			"project_name", p.Name,
			"error", err)
		return "", err
	}

	return p.executeCommand(cmd)
}

func (p *ComposeProject) prepareCommand(command string, args []string) (*exec.Cmd, error) {
	// Build docker compose command
	commandArgs := []string{
		"--host", p.Config.DockerHost,
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
		"command", p.Config.DockerCommand,
		"args", commandArgs,
		"working_dir", p.WorkingDir)

	// Create command
	cmd := exec.Command(p.Config.DockerCommand, commandArgs...)
	cmd.Dir = p.WorkingDir

	// Inject variables if provided
	if len(p.Variables) > 0 {
		// Start with existing environment and append/override with user variables
		cmd.Env = append(os.Environ(), p.Variables...)
		slog.Debug("Injecting variables",
			"project_name", p.Name,
			"var_count", len(p.Variables))
	}

	return cmd, nil
}

func (p *ComposeProject) executeCommand(cmd *exec.Cmd) (string, error) {
	out, err := cmd.CombinedOutput()
	output := string(out)
	if err != nil {
		slog.Error("Service operation failed",
			"layer", "docker_compose",
			"operation", "docker_compose_execute",
			"project_name", p.Name,
			"error", err,
			"output", output)
		return "", err
	}
	return output, nil
}

func (p *ComposeProject) executeCommandStreaming(cmd *exec.Cmd, outputChan chan<- string) error {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		slog.Error("Service operation failed",
			"layer", "docker_compose",
			"operation", "docker_compose_stream",
			"command", cmd.String(),
			"error", err)
		return err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		slog.Error("Service operation failed",
			"layer", "docker_compose",
			"operation", "docker_compose_stream",
			"command", cmd.String(),
			"error", err)
		return err
	}

	// Start the command
	err = cmd.Start()
	if err != nil {
		slog.Error("Service operation failed",
			"layer", "docker_compose",
			"operation", "docker_compose_stream",
			"command", cmd.String(),
			"error", err)
		return err
	}

	// Use a WaitGroup to ensure all goroutines complete before returning
	var wg sync.WaitGroup

	// Stream stdout
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			msg := map[string]string{
				"type":    "docker",
				"message": scanner.Text(),
			}
			if jsonMsg, err := json.Marshal(msg); err == nil {
				outputChan <- string(jsonMsg)
			}
		}
	}()

	// Stream stderr
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			msg := map[string]string{
				"type":    "docker",
				"message": scanner.Text(),
			}
			if jsonMsg, err := json.Marshal(msg); err == nil {
				outputChan <- string(jsonMsg)
			}
		}
	}()

	// Wait for command to finish
	cmdErr := cmd.Wait()

	// Wait for all goroutines to finish reading output before checking for errors
	// This ensures all output is processed even if the command failed
	wg.Wait()

	if cmdErr != nil {
		slog.Error("Service operation failed",
			"layer", "docker_compose",
			"operation", "docker_compose_stream",
			"command", cmd.String(),
			"error", cmdErr)
		return cmdErr
	}

	slog.Debug("Docker Compose command completed successfully")

	return nil
}

func (p *ComposeProject) executeCommandPiping(cmd *exec.Cmd) error {
	// Inherit stdout and stderr for direct piping to terminal
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	slog.Debug("Executing Docker Compose command with direct piping",
		"project_name", p.Name,
		"command", cmd.String(),
		"working_dir", p.WorkingDir)

	err := cmd.Run()
	if err != nil {
		slog.Error("Service operation failed",
			"layer", "docker_compose",
			"operation", "docker_compose_piping",
			"project_name", p.Name,
			"command", cmd.String(),
			"error", err)
		return err
	}

	return nil
}

func (p *ComposeProject) commandUp() (*exec.Cmd, error) {
	cmd, err := p.prepareCommand("up", []string{"--detach", "--wait", "--quiet-pull", "--no-color", "--remove-orphans"})
	if err != nil {
		slog.Error("Service operation failed",
			"layer", "docker_compose",
			"operation", "docker_compose_up",
			"project_name", p.Name,
			"error", err)
		return nil, err
	}

	return cmd, nil
}

func (p *ComposeProject) commandDown() (*exec.Cmd, error) {
	cmd, err := p.prepareCommand("down", []string{"--remove-orphans"})
	if err != nil {
		slog.Error("Service operation failed",
			"layer", "docker_compose",
			"operation", "docker_compose_down",
			"project_name", p.Name,
			"error", err)
		return nil, err
	}

	return cmd, nil
}

func (p *ComposeProject) commandLogs() (*exec.Cmd, error) {
	// TODO: Implement color configuration that supports both CLI and web UI
	// Currently hardcoded to --no-color, but should be configurable.
	// Cannot import cmd/output here due to import cycle (cmd/output imports services).
	// Options:
	// 1. Add color config to ComposeProject struct/Config
	// 2. Pass color preference as parameter to commandLogs()
	// 3. Create a shared config package that both can import
	cmd, err := p.prepareCommand("logs", []string{"--no-color", "--follow"})
	if err != nil {
		slog.Error("Service operation failed",
			"layer", "docker_compose",
			"operation", "docker_compose_logs",
			"project_name", p.Name,
			"error", err)
		return nil, err
	}

	return cmd, nil
}

func (p *ComposeProject) commandConfig() (*exec.Cmd, error) {
	cmd, err := p.prepareCommand("config", []string{})
	if err != nil {
		slog.Error("Service operation failed",
			"layer", "docker_compose",
			"operation", "docker_compose_config",
			"project_name", p.Name,
			"error", err)
		return nil, err
	}

	return cmd, nil
}

func (p *ComposeProject) commandPs() (*exec.Cmd, error) {
	cmd, err := p.prepareCommand("ps", []string{"--format", "json"})
	if err != nil {
		slog.Error("Service operation failed",
			"layer", "docker_compose",
			"operation", "docker_compose_ps",
			"project_name", p.Name,
			"error", err)
		return nil, err
	}

	return cmd, nil
}

func (p *ComposeProject) Status() (*ComposeStatus, error) {
	cmd, err := p.commandPs()
	if err != nil {
		return nil, err
	}

	output, err := p.executeCommand(cmd)
	if err != nil {
		slog.Error("Service operation failed",
			"layer", "docker_compose",
			"operation", "docker_compose_status",
			"project_name", p.Name,
			"error", err)
		return nil, err
	}

	var containers []ContainerInfo
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		var container ContainerInfo
		if err := json.Unmarshal([]byte(line), &container); err != nil {
			slog.Error("Failed to parse container JSON",
				"project_name", p.Name,
				"line", line,
				"error", err)
			continue
		}
		containers = append(containers, container)
	}

	// Determine overall project status
	projectStatus := "stopped"
	uptime := ""
	if len(containers) > 0 {
		runningCount := 0
		for _, container := range containers {
			if container.State == "running" {
				runningCount++
				if uptime == "" {
					uptime = strings.TrimSuffix(container.RunningFor, " ago")
				}
			}
		}

		if runningCount == len(containers) {
			projectStatus = "running"
		} else if runningCount > 0 {
			projectStatus = "partial"
		}
	}

	return &ComposeStatus{
		Status:     projectStatus,
		Containers: containers,
		Uptime:     uptime,
	}, nil
}
