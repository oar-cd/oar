package services

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// StreamMessage represents a message in the streaming output
type StreamMessage struct {
	Type    string `json:"type"`    // "stdout", "stderr", "info", "success", "error"
	Content string `json:"content"` // the actual message content
}

type ComposeProjectStatus int

const (
	ComposeProjectStatusUnknown ComposeProjectStatus = iota
	ComposeProjectStatusRunning
	ComposeProjectStatusStopped
	ComposeProjectStatusFailed
)

func (s ComposeProjectStatus) String() string {
	switch s {
	case ComposeProjectStatusRunning:
		return "running"
	case ComposeProjectStatusStopped:
		return "stopped"
	case ComposeProjectStatusFailed:
		return "failed"
	case ComposeProjectStatusUnknown:
		return "unknown"
	default:
		return "unknown"
	}
}

func ParseComposeProjectStatus(s string) (ComposeProjectStatus, error) {
	switch s {
	case "running":
		return ComposeProjectStatusRunning, nil
	case "stopped":
		return ComposeProjectStatusStopped, nil
	case "failed":
		return ComposeProjectStatusFailed, nil
	case "unknown":
		return ComposeProjectStatusUnknown, nil
	default:
		return ComposeProjectStatusUnknown, fmt.Errorf("invalid compose project status: %q", s)
	}
}

type ContainerInfo struct {
	Service    string `json:"Service"`
	Name       string `json:"Name"`
	State      string `json:"State"`
	Status     string `json:"Status"`
	RunningFor string `json:"RunningFor"`
	ExitCode   int    `json:"ExitCode"`
}

type ComposeStatus struct {
	Status     ComposeProjectStatus
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

func (p *ComposeProject) Up(startServices bool) (string, string, error) {
	cmd := p.commandUp(startServices)
	stdout, stderr, err := p.executeCommand(cmd)
	if err != nil {
		return "", "", err
	}
	return stdout, stderr, nil
}

func (p *ComposeProject) UpStreaming(startServices bool, outputChan chan<- StreamMessage) error {
	cmd := p.commandUp(startServices)
	return p.executeCommandStreaming(cmd, outputChan)
}

func (p *ComposeProject) UpPiping(startServices bool) error {
	cmd := p.commandUp(startServices)
	return p.executeCommandPiping(cmd)
}

func (p *ComposeProject) Down(removeVolumes bool) (string, string, error) {
	cmd := p.commandDown(removeVolumes)
	stdout, stderr, err := p.executeCommand(cmd)
	if err != nil {
		return "", "", err
	}
	return stdout, stderr, nil
}

func (p *ComposeProject) DownStreaming(outputChan chan<- StreamMessage) error {
	cmd := p.commandDown(false)
	return p.executeCommandStreaming(cmd, outputChan)
}

func (p *ComposeProject) DownPiping() error {
	cmd := p.commandDown(false)
	return p.executeCommandPiping(cmd)
}

func (p *ComposeProject) Logs() (string, string, error) {
	cmd := p.commandLogs(false) // No follow for static logs
	stdout, stderr, err := p.executeCommand(cmd)
	if err != nil {
		return "", "", err
	}
	return stdout, stderr, nil
}

func (p *ComposeProject) LogsPiping() error {
	cmd := p.commandLogs(true) // Follow for CLI streaming
	return p.executeCommandPiping(cmd)
}

func (p *ComposeProject) GetConfig() (string, string, error) {
	cmd := p.commandConfig()
	stdout, stderr, err := p.executeCommand(cmd)
	if err != nil {
		return "", "", err
	}
	return stdout, stderr, nil
}

func (p *ComposeProject) Pull() (string, string, error) {
	cmd := p.commandPull()
	stdout, stderr, err := p.executeCommand(cmd)
	if err != nil {
		return "", "", err
	}
	return stdout, stderr, nil
}

func (p *ComposeProject) Build() (string, string, error) {
	cmd := p.commandBuild()
	stdout, stderr, err := p.executeCommand(cmd)
	if err != nil {
		return "", "", err
	}
	return stdout, stderr, nil
}

func (p *ComposeProject) prepareCommand(command string, args []string) *exec.Cmd {
	// Build docker compose command
	commandArgs := []string{
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
		"command", "docker",
		"args", commandArgs,
		"project_name", p.Name)

	// Create command
	cmd := exec.Command("docker", commandArgs...)
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

func (p *ComposeProject) executeCommand(cmd *exec.Cmd) (stdout string, stderr string, err error) {
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err = cmd.Run()
	stdout = stdoutBuf.String()
	stderr = stderrBuf.String()

	if err != nil {
		slog.Error("Service operation failed",
			"layer", "docker_compose",
			"operation", "docker_compose_execute",
			"project_name", p.Name,
			"error", err,
			"stdout", stdout,
			"stderr", stderr)
		return stdout, stderr, err
	}
	return stdout, stderr, nil
}

func (p *ComposeProject) executeCommandStreaming(cmd *exec.Cmd, outputChan chan<- StreamMessage) error {
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
		defer func() {
			if err := stdout.Close(); err != nil {
				slog.Debug("Failed to close stdout pipe", "error", err)
			}
		}()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			select {
			case outputChan <- StreamMessage{Type: "stdout", Content: scanner.Text()}:
			default:
				return
			}
		}
	}()

	// Stream stderr
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			if err := stderr.Close(); err != nil {
				slog.Debug("Failed to close stderr pipe", "error", err)
			}
		}()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			rawLine := scanner.Text()
			// Parse Docker Compose structured logs to extract just the message
			parsedContent := ParseComposeLogLine(rawLine)
			select {
			case outputChan <- StreamMessage{Type: "stderr", Content: parsedContent}:
			default:
				return
			}
		}
	}()

	// Wait for command to finish
	cmdErr := cmd.Wait()

	// Wait for all goroutines to finish reading output before checking for errors
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

func (p *ComposeProject) commandUp(startServices bool) *exec.Cmd {
	args := []string{"--detach", "--quiet-pull", "--quiet-build", "--remove-orphans"}
	if !startServices {
		args = append(args, "--no-start")
	}
	return p.prepareCommand("up", args)
}

func (p *ComposeProject) commandDown(removeVolumes bool) *exec.Cmd {
	args := []string{"--remove-orphans"}
	if removeVolumes {
		args = append(args, "--volumes")
	}
	return p.prepareCommand("down", args)
}

func (p *ComposeProject) commandLogs(follow bool) *exec.Cmd {
	args := []string{}
	if follow {
		args = append(args, "--follow")
	}
	return p.prepareCommand("logs", args)
}

func (p *ComposeProject) commandConfig() *exec.Cmd {
	return p.prepareCommand("config", []string{})
}

func (p *ComposeProject) commandPull() *exec.Cmd {
	return p.prepareCommand("pull", []string{})
}

func (p *ComposeProject) commandBuild() *exec.Cmd {
	return p.prepareCommand("build", []string{})
}

func (p *ComposeProject) commandPs() *exec.Cmd {
	return p.prepareCommand("ps", []string{"--format", "json"})
}

func (p *ComposeProject) Status() (*ComposeStatus, error) {
	cmd := p.commandPs()

	stdout, stderr, err := p.executeCommand(cmd)
	if err != nil {
		slog.Error("Service operation failed",
			"layer", "docker_compose",
			"operation", "docker_compose_status",
			"project_name", p.Name,
			"error", err,
			"stderr", stderr)
		return nil, err
	}

	var containers []ContainerInfo
	lines := strings.Split(strings.TrimSpace(stdout), "\n")

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
	projectStatus := ComposeProjectStatusStopped
	uptime := ""
	if len(containers) > 0 {
		runningCount := 0
		totalRelevantContainers := 0

		for _, container := range containers {
			// Skip containers that have legitimately exited with success (e.g., init containers)
			if container.State == "exited" && container.ExitCode == 0 {
				continue
			}

			totalRelevantContainers++
			if container.State == "running" {
				runningCount++
				if uptime == "" {
					uptime = strings.TrimSuffix(container.RunningFor, " ago")
				}
			}
		}

		if totalRelevantContainers == 0 {
			// All containers are successfully exited init containers - we can't determine status
			projectStatus = ComposeProjectStatusUnknown
		} else if runningCount == totalRelevantContainers {
			projectStatus = ComposeProjectStatusRunning
		} else if runningCount > 0 {
			projectStatus = ComposeProjectStatusFailed
		}
	}

	return &ComposeStatus{
		Status:     projectStatus,
		Containers: containers,
		Uptime:     uptime,
	}, nil
}

// ParseComposeLogLine parses Docker Compose structured log format and extracts the msg value
// Format: time="..." level="..." msg="..." [other fields]
// Returns the msg value if found, otherwise returns the original line
func ParseComposeLogLine(line string) string {
	// Regex to match msg="..." handling escaped quotes
	msgRegex := regexp.MustCompile(`msg="((?:[^"\\]|\\.)*)"\s*(?:\s|$)`)
	matches := msgRegex.FindStringSubmatch(line)
	if len(matches) > 1 {
		// Unescape the extracted message
		msg := matches[1]
		// First replace escaped quotes, then escaped backslashes (order matters)
		msg = strings.ReplaceAll(msg, `\"`, `"`)
		msg = strings.ReplaceAll(msg, `\\`, `\`)
		return msg
	}
	// If no msg field found, return original line
	return line
}
