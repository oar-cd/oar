package services

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
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
	cmd := p.commandUp(context.Background())
	return p.executeCommand(cmd)
}

func (p *ComposeProject) UpStreaming(outputChan chan<- string) error {
	cmd := p.commandUp(context.Background())
	return p.executeCommandStreaming(context.Background(), cmd, outputChan)
}

func (p *ComposeProject) UpPiping() error {
	cmd := p.commandUp(context.Background())
	return p.executeCommandPiping(cmd)
}

func (p *ComposeProject) Down() (string, error) {
	cmd := p.commandDown(context.Background())
	return p.executeCommand(cmd)
}

func (p *ComposeProject) DownStreaming(outputChan chan<- string) error {
	cmd := p.commandDown(context.Background())
	return p.executeCommandStreaming(context.Background(), cmd, outputChan)
}

func (p *ComposeProject) DownPiping() error {
	cmd := p.commandDown(context.Background())
	return p.executeCommandPiping(cmd)
}

func (p *ComposeProject) Logs() (string, error) {
	cmd := p.commandLogs(context.Background())
	return p.executeCommand(cmd)
}

func (p *ComposeProject) LogsStreaming(ctx context.Context, outputChan chan<- string) error {
	cmd := p.commandLogs(ctx)
	return p.executeCommandStreaming(ctx, cmd, outputChan)
}

func (p *ComposeProject) LogsPiping() error {
	cmd := p.commandLogs(context.Background())
	return p.executeCommandPiping(cmd)
}

func (p *ComposeProject) GetConfig() (string, error) {
	cmd := p.commandConfig(context.Background())
	return p.executeCommand(cmd)
}

func (p *ComposeProject) prepareCommand(ctx context.Context, command string, args []string) *exec.Cmd {
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
	cmd := exec.CommandContext(ctx, "docker", commandArgs...)
	// Do not set cmd.Dir to avoid Docker resolving container paths as host paths.
	// The compose files are already specified with absolute paths via --file flags.

	// Set up process group to ensure child processes are also terminated on cancellation
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

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

func (p *ComposeProject) executeCommandStreaming(ctx context.Context, cmd *exec.Cmd, outputChan chan<- string) error {
	// Set up command pipes and start the process
	stdout, stderr, err := p.setupCommandPipes(cmd)
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		slog.Error("Service operation failed",
			"layer", "docker_compose",
			"operation", "docker_compose_stream",
			"command", cmd.String(),
			"error", err)
		return err
	}

	// Log process group information
	p.logProcessGroupInfo(cmd)

	// Set up process monitoring and termination
	cmdDone := make(chan error, 1)
	go func() {
		cmdDone <- cmd.Wait()
	}()

	// Start process termination monitor
	go p.monitorProcessTermination(ctx, cmd, cmdDone)

	// Start output streaming
	var wg sync.WaitGroup
	p.streamOutput(ctx, stdout, stderr, outputChan, &wg)

	// Wait for command completion and handle results
	return p.handleCommandCompletion(ctx, cmd, cmdDone, &wg)
}

// setupCommandPipes sets up stdout and stderr pipes for the command
func (p *ComposeProject) setupCommandPipes(cmd *exec.Cmd) (stdout, stderr io.ReadCloser, err error) {
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		slog.Error("Service operation failed",
			"layer", "docker_compose",
			"operation", "docker_compose_stream",
			"command", cmd.String(),
			"error", err)
		return nil, nil, err
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		slog.Error("Service operation failed",
			"layer", "docker_compose",
			"operation", "docker_compose_stream",
			"command", cmd.String(),
			"error", err)
		return nil, nil, err
	}

	return stdoutPipe, stderrPipe, nil
}

// logProcessGroupInfo logs information about the process group
func (p *ComposeProject) logProcessGroupInfo(cmd *exec.Cmd) {
	if cmd.Process != nil {
		pgid, err := syscall.Getpgid(cmd.Process.Pid)
		if err != nil {
			slog.Debug("Failed to get process group ID",
				"project_name", p.Name,
				"parent_pid", cmd.Process.Pid,
				"error", err)
		} else {
			groupMembers := p.getProcessGroupMembers(pgid)
			slog.Debug("Process started with new process group",
				"project_name", p.Name,
				"parent_pid", cmd.Process.Pid,
				"process_group_id", pgid,
				"group_members", groupMembers)
		}
	}
}

// getProcessGroupMembers returns all PIDs that belong to the same process group
func (p *ComposeProject) getProcessGroupMembers(pgid int) []int {
	var pids []int

	// Read /proc to find all processes with the same PGID
	procEntries, err := os.ReadDir("/proc")
	if err != nil {
		slog.Debug("Failed to read /proc directory",
			"project_name", p.Name,
			"error", err)
		return pids
	}

	for _, entry := range procEntries {
		if !entry.IsDir() {
			continue
		}

		// Check if directory name is a PID (numeric)
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}

		// Read the process's stat file to get PGID
		statusPath := fmt.Sprintf("/proc/%d/stat", pid)
		statusData, err := os.ReadFile(statusPath)
		if err != nil {
			continue // Process might have exited
		}

		// Parse the stat file - PGID is the 5th field (index 4)
		fields := strings.Fields(string(statusData))
		if len(fields) < 5 {
			continue
		}

		processPgid, err := strconv.Atoi(fields[4])
		if err != nil {
			continue
		}

		if processPgid == pgid {
			pids = append(pids, pid)
		}
	}

	return pids
}

// streamOutput starts goroutines to stream stdout and stderr
func (p *ComposeProject) streamOutput(
	ctx context.Context,
	stdout, stderr io.ReadCloser,
	outputChan chan<- string,
	wg *sync.WaitGroup,
) {
	// Stream stdout
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			if err := stdout.Close(); err != nil {
				slog.Debug("Failed to close stdout pipe",
					"project_name", p.Name,
					"error", err)
			}
		}()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				slog.Debug("Context cancelled, stopping stdout streaming",
					"project_name", p.Name)
				return
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
		defer func() {
			if err := stderr.Close(); err != nil {
				slog.Debug("Failed to close stderr pipe",
					"project_name", p.Name,
					"error", err)
			}
		}()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				slog.Debug("Context cancelled, stopping stderr streaming",
					"project_name", p.Name)
				return
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
}

// handleCommandCompletion waits for command completion and handles results
func (p *ComposeProject) handleCommandCompletion(
	ctx context.Context,
	cmd *exec.Cmd,
	cmdDone <-chan error,
	wg *sync.WaitGroup,
) error {
	// Wait for command to finish
	cmdErr := <-cmdDone

	// Wait for all goroutines to finish reading output before checking for errors
	// This ensures all output is processed even if the command failed
	wg.Wait()

	if cmdErr != nil {
		// Check if error is due to context cancellation
		if ctx.Err() != nil {
			slog.Info("Docker Compose command cancelled by context",
				"project_name", p.Name,
				"command", cmd.String())
			return ctx.Err()
		}
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

// monitorProcessTermination monitors context cancellation and terminates the process group
func (p *ComposeProject) monitorProcessTermination(ctx context.Context, cmd *exec.Cmd, cmdDone <-chan error) {
	select {
	case <-ctx.Done():
		p.terminateProcessGroup(ctx, cmd, cmdDone)
	case <-cmdDone:
		// Command completed normally, nothing to do
		return
	}
}

// terminateProcessGroup handles graceful and forced termination of the process group
func (p *ComposeProject) terminateProcessGroup(ctx context.Context, cmd *exec.Cmd, cmdDone <-chan error) {
	if cmd.Process == nil {
		return
	}

	pgid, _ := syscall.Getpgid(cmd.Process.Pid)
	groupMembers := p.getProcessGroupMembers(pgid)
	slog.Debug("Context cancelled, terminating process group",
		"project_name", p.Name,
		"parent_pid", cmd.Process.Pid,
		"process_group_id", pgid,
		"group_members", groupMembers)

	// First try graceful termination
	if p.sendSignalToProcessGroup(cmd.Process.Pid, syscall.SIGTERM, pgid, groupMembers) {
		return // Process already exited
	}

	// Wait for graceful termination with timeout
	timeout := p.calculateTerminationTimeout(ctx)

	slog.Debug("Waiting for graceful termination",
		"project_name", p.Name,
		"waiting_duration", timeout)

	waitStart := time.Now()
	select {
	case <-cmdDone:
		waitDuration := time.Since(waitStart)
		slog.Debug("Process terminated gracefully after SIGTERM",
			"project_name", p.Name,
			"actual_wait_time", waitDuration)
		return
	case <-time.After(timeout):
		waitDuration := time.Since(waitStart)
		slog.Debug("Process did not terminate within timeout, forcing kill",
			"project_name", p.Name,
			"configured_timeout", timeout,
			"actual_wait_time", waitDuration)
	}

	// Force kill if still running
	if cmd.Process != nil {
		pgid, _ := syscall.Getpgid(cmd.Process.Pid)
		groupMembers := p.getProcessGroupMembers(pgid)
		p.sendSignalToProcessGroup(cmd.Process.Pid, syscall.SIGKILL, pgid, groupMembers)
	}
}

// sendSignalToProcessGroup sends a signal to the process group and handles errors
// Returns true if process already exited (ESRCH), false otherwise
func (p *ComposeProject) sendSignalToProcessGroup(pid int, signal syscall.Signal, pgid int, groupMembers []int) bool {
	signalName := "SIGTERM"
	if signal == syscall.SIGKILL {
		signalName = "SIGKILL"
	}

	if err := syscall.Kill(-pid, signal); err != nil {
		slog.Debug(fmt.Sprintf("Failed to send %s to process group", signalName),
			"project_name", p.Name,
			"parent_pid", pid,
			"process_group_id", pgid,
			"group_members", groupMembers,
			"error", err)

		// If process doesn't exist, it already exited
		if err == syscall.ESRCH {
			if signal == syscall.SIGTERM {
				slog.Debug("Process already exited, skipping termination logic",
					"project_name", p.Name,
					"parent_pid", pid)
			} else {
				slog.Debug("Process already exited during force kill attempt",
					"project_name", p.Name,
					"parent_pid", pid)
			}
			return true
		}
		return false
	}

	slog.Debug(fmt.Sprintf("Sent %s to process group", signalName),
		"project_name", p.Name,
		"parent_pid", pid,
		"process_group_id", pgid,
		"group_members", groupMembers)
	return false
}

// calculateTerminationTimeout calculates the timeout for graceful termination
func (p *ComposeProject) calculateTerminationTimeout(ctx context.Context) time.Duration {
	timeout := 20 * time.Second
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		slog.Debug("Context has deadline, calculating timeout",
			"project_name", p.Name,
			"context_deadline", deadline,
			"remaining_time", remaining,
			"default_timeout", timeout)
		if remaining < timeout && remaining > 0 {
			timeout = remaining
			slog.Debug("Using shorter context deadline as timeout",
				"project_name", p.Name,
				"final_timeout", timeout)
		} else {
			slog.Debug("Context deadline is longer than default, using default timeout",
				"project_name", p.Name,
				"final_timeout", timeout)
		}
	} else {
		slog.Debug("Context has no deadline, using default timeout",
			"project_name", p.Name,
			"final_timeout", timeout)
	}

	return timeout
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

func (p *ComposeProject) commandUp(ctx context.Context) *exec.Cmd {
	return p.prepareCommand(ctx, "up", []string{"--detach", "--wait", "--quiet-pull", "--remove-orphans"})
}

func (p *ComposeProject) commandDown(ctx context.Context) *exec.Cmd {
	return p.prepareCommand(ctx, "down", []string{"--remove-orphans"})
}

func (p *ComposeProject) commandLogs(ctx context.Context) *exec.Cmd {
	return p.prepareCommand(ctx, "logs", []string{"--follow"})
}

func (p *ComposeProject) commandConfig(ctx context.Context) *exec.Cmd {
	return p.prepareCommand(ctx, "config", []string{})
}

func (p *ComposeProject) commandPs(ctx context.Context) *exec.Cmd {
	return p.prepareCommand(ctx, "ps", []string{"--format", "json"})
}

func (p *ComposeProject) Status() (*ComposeStatus, error) {
	cmd := p.commandPs(context.Background())

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
