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
	cmd := p.commandUp()
	return p.executeCommand(cmd)
}

func (p *ComposeProject) UpStreaming(outputChan chan<- string) error {
	cmd := p.commandUp()
	return p.executeCommandStreaming(cmd, outputChan)
}

func (p *ComposeProject) UpPiping() error {
	cmd := p.commandUp()
	return p.executeCommandPiping(cmd)
}

func (p *ComposeProject) Down() (string, error) {
	cmd := p.commandDown()
	return p.executeCommand(cmd)
}

func (p *ComposeProject) DownStreaming(outputChan chan<- string) error {
	cmd := p.commandDown()
	return p.executeCommandStreaming(cmd, outputChan)
}

func (p *ComposeProject) DownPiping() error {
	cmd := p.commandDown()
	return p.executeCommandPiping(cmd)
}

func (p *ComposeProject) Logs() (string, error) {
	cmd := p.commandLogs()
	return p.executeCommand(cmd)
}

func (p *ComposeProject) LogsStreaming(outputChan chan<- string) error {
	cmd := p.commandLogs()
	return p.executeCommandStreaming(cmd, outputChan)
}

func (p *ComposeProject) LogsPiping() error {
	cmd := p.commandLogs()
	return p.executeCommandPiping(cmd)
}

func (p *ComposeProject) GetConfig() (string, error) {
	cmd := p.commandConfig()
	return p.executeCommand(cmd)
}

func (p *ComposeProject) prepareCommand(command string, args []string) *exec.Cmd {
	// Build docker compose command
	commandArgs := []string{
		"--host", p.Config.DockerHost,
		"compose",
		"--progress", "plain",
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
		"project_name", p.Name)

	// Create command
	cmd := exec.Command(p.Config.DockerCommand, commandArgs...)
	// Do not set cmd.Dir to avoid Docker resolving container paths as host paths.
	// The compose files are already specified with absolute paths via --file flags.

	// Disable color output to simplify parsing logs and status
	cmd.Env = append(os.Environ(), "NO_COLOR=1")

	// Inject variables if provided
	if len(p.Variables) > 0 {
		// Start with existing environment and append/override with user variables
		cmd.Env = append(cmd.Env, p.Variables...)
		slog.Debug("Injecting variables",
			"project_name", p.Name,
			"var_count", len(p.Variables))
	}

	return cmd
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
			select {
			case outputChan <- scanner.Text():
			default:
				// Channel is full or closed (likely client disconnected), skip this message
				slog.Debug("Dropped Docker stdout message, channel unavailable",
					"project_name", p.Name,
					"message_type", "stdout")
				return
			}
		}
	}()

	// Stream stderr
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			select {
			case outputChan <- scanner.Text():
			default:
				// Channel is full or closed (likely client disconnected), skip this message
				slog.Debug("Dropped Docker stderr message, channel unavailable",
					"project_name", p.Name,
					"message_type", "stderr")
				return
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

func (p *ComposeProject) commandUp() *exec.Cmd {
	return p.prepareCommand("up", []string{"--detach", "--wait", "--quiet-pull", "--no-color", "--remove-orphans"})
}

func (p *ComposeProject) commandDown() *exec.Cmd {
	return p.prepareCommand("down", []string{"--remove-orphans"})
}

func (p *ComposeProject) commandLogs() *exec.Cmd {
	return p.prepareCommand("logs", []string{"--follow"})
}

func (p *ComposeProject) commandConfig() *exec.Cmd {
	return p.prepareCommand("config", []string{})
}

func (p *ComposeProject) commandPs() *exec.Cmd {
	return p.prepareCommand("ps", []string{"--format", "json"})
}

func (p *ComposeProject) Status() (*ComposeStatus, error) {
	cmd := p.commandPs()

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
