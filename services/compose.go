package services

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/compose-spec/compose-go/v2/cli"
	"github.com/compose-spec/compose-go/v2/dotenv"
	"github.com/compose-spec/compose-go/v2/loader"
	"github.com/compose-spec/compose-go/v2/types"
	"gopkg.in/yaml.v3"
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
	// Project holds a reference to the domain project for cache directory access
	Project *Project
}

// Ensure ComposeProject implements ComposeProjectInterface
var _ ComposeProjectInterface = (*ComposeProject)(nil)

// getProcessedComposeFileName returns the filename for a processed compose file
// Example: "docker-compose.yml" -> "docker-compose.processed.yaml"
func (p *ComposeProject) getProcessedComposeFileName(originalFileName string) string {
	ext := filepath.Ext(originalFileName)
	baseName := strings.TrimSuffix(originalFileName, ext)
	return baseName + ".processed.yaml"
}

// getCachedComposeFiles returns paths to cached processed compose files,
// always regenerating them (no cache validation)
func (p *ComposeProject) getCachedComposeFiles(project *Project) ([]string, error) {
	if len(p.ComposeFiles) == 0 {
		return nil, fmt.Errorf("no compose files to process")
	}

	if p.Config.DataDir == "" {
		return nil, fmt.Errorf("data directory not configured, cannot process volume paths")
	}

	// Get cache directory from project
	cacheDir, err := project.CacheDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get cache directory: %w", err)
	}

	// Calculate host equivalent directory for volume path conversion
	// Container: /data/projects/abc123/git -> Host: ./data/projects/abc123/git (relative to compose.yaml location)
	relativePath := strings.TrimPrefix(p.WorkingDir, p.Config.DataDir)
	hostWorkingDir := filepath.Join(".", "data") + relativePath

	var cachedFiles []string

	for _, composeFile := range p.ComposeFiles {
		originalPath := filepath.Join(p.WorkingDir, composeFile)
		cachedFileName := p.getProcessedComposeFileName(composeFile)
		cachedFilePath := filepath.Join(cacheDir, cachedFileName)

		// Always regenerate - no cache validation
		slog.Debug("Processing compose file",
			"original", originalPath,
			"cached", cachedFilePath)

		// Load and process the compose file
		processedProject, err := p.loadAndProcessComposeFile(originalPath, hostWorkingDir)
		if err != nil {
			return nil, fmt.Errorf("failed to process compose file %s: %w", originalPath, err)
		}

		// Write to cache
		if err := p.writeCachedComposeFile(processedProject, cachedFilePath, composeFile); err != nil {
			return nil, fmt.Errorf("failed to write cached compose file %s: %w", cachedFilePath, err)
		}

		cachedFiles = append(cachedFiles, cachedFilePath)
	}

	return cachedFiles, nil
}

// processVolumePaths modifies volume paths in the project to use absolute host paths
func (p *ComposeProject) processVolumePaths(
	project *types.Project,
	hostWorkingDir, containerWorkingDir string,
) (*types.Project, error) {
	for serviceName, service := range project.Services {
		for i, volume := range service.Volumes {
			if volume.Source != "" && strings.HasPrefix(volume.Source, ".") {
				// Convert relative path to absolute host path
				var absoluteHostPath string
				if volume.Source == "." {
					absoluteHostPath = hostWorkingDir
				} else if strings.HasPrefix(volume.Source, "./") {
					relativePart := strings.TrimPrefix(volume.Source, "./")
					absoluteHostPath = filepath.Join(hostWorkingDir, relativePart)
				}

				if absoluteHostPath != "" {
					slog.Debug("Converting relative volume path to absolute",
						"service", serviceName,
						"original", volume.Source,
						"converted", absoluteHostPath)
					service.Volumes[i].Source = absoluteHostPath
				}
			}
		}
		project.Services[serviceName] = service
	}

	return project, nil
}

// loadAndProcessComposeFile loads a compose file and processes its volume paths
func (p *ComposeProject) loadAndProcessComposeFile(originalPath, hostWorkingDir string) (*types.Project, error) {
	// Load the compose project using compose-go
	workingDir := filepath.Dir(originalPath)
	envPath := filepath.Join(workingDir, ".env")

	// Load .env file variables if it exists (similar to discovery.go)
	envMap := make(map[string]string)
	if _, err := os.Stat(envPath); err == nil {
		envVars, err := dotenv.Read(envPath)
		if err == nil {
			// Add .env variables to map
			for key, value := range envVars {
				envMap[key] = value
			}
		}
	}

	// Add/override with project variables (explicit precedence: project variables win)
	for _, variable := range p.Variables {
		parts := strings.SplitN(variable, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	// Convert map back to slice
	var envSlice []string
	for key, value := range envMap {
		envSlice = append(envSlice, key+"="+value)
	}

	// Create project options
	options, err := cli.NewProjectOptions([]string{originalPath},
		cli.WithWorkingDirectory(workingDir),
		cli.WithEnv(envSlice),
		cli.WithLoadOptions(func(o *loader.Options) {
			o.SkipValidation = true
			o.SkipConsistencyCheck = true
		}))
	if err != nil {
		return nil, fmt.Errorf("failed to create project options: %w", err)
	}

	// Load the project
	ctx := context.Background()
	project, err := options.LoadProject(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load compose project: %w", err)
	}

	// Process volume paths
	return p.processVolumePaths(project, hostWorkingDir, workingDir)
}

// writeCachedComposeFile writes the processed project to the cache directory
func (p *ComposeProject) writeCachedComposeFile(project *types.Project, cachedFilePath, originalFileName string) error {
	// Create the cached file
	file, err := os.Create(cachedFilePath)
	if err != nil {
		return fmt.Errorf("failed to create cached file %s: %w", cachedFilePath, err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			slog.Error("Failed to close cached file", "file", cachedFilePath, "error", closeErr)
		}
	}()

	// Encode project directly to YAML
	encoder := yaml.NewEncoder(file)
	defer func() {
		if closeErr := encoder.Close(); closeErr != nil {
			slog.Error("Failed to close YAML encoder", "file", cachedFilePath, "error", closeErr)
		}
	}()

	if err := encoder.Encode(project); err != nil {
		return fmt.Errorf("failed to encode project to YAML: %w", err)
	}

	slog.Debug("Created cached compose file",
		"original", originalFileName,
		"cached", cachedFilePath,
		"project", p.Name)

	return nil
}

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
		Project:      p,
	}
}

func (p *ComposeProject) Up() (string, error) {
	cmd, err := p.commandUp()
	if err != nil {
		return "", err
	}
	return p.executeCommand(cmd)
}

func (p *ComposeProject) UpStreaming(outputChan chan<- string) error {
	cmd, err := p.commandUp()
	if err != nil {
		return err
	}
	return p.executeCommandStreaming(cmd, outputChan)
}

func (p *ComposeProject) UpPiping() error {
	cmd, err := p.commandUp()
	if err != nil {
		return err
	}
	return p.executeCommandPiping(cmd)
}

func (p *ComposeProject) Down() (string, error) {
	cmd, err := p.commandDown()
	if err != nil {
		return "", err
	}
	return p.executeCommand(cmd)
}

func (p *ComposeProject) DownStreaming(outputChan chan<- string) error {
	cmd, err := p.commandDown()
	if err != nil {
		return err
	}
	return p.executeCommandStreaming(cmd, outputChan)
}

func (p *ComposeProject) DownPiping() error {
	cmd, err := p.commandDown()
	if err != nil {
		return err
	}
	return p.executeCommandPiping(cmd)
}

func (p *ComposeProject) Logs() (string, error) {
	cmd, err := p.commandLogs()
	if err != nil {
		return "", err
	}
	return p.executeCommand(cmd)
}

func (p *ComposeProject) LogsStreaming(outputChan chan<- string) error {
	cmd, err := p.commandLogs()
	if err != nil {
		return err
	}
	return p.executeCommandStreaming(cmd, outputChan)
}

func (p *ComposeProject) LogsPiping() error {
	cmd, err := p.commandLogs()
	if err != nil {
		return err
	}
	return p.executeCommandPiping(cmd)
}

func (p *ComposeProject) GetConfig() (string, error) {
	cmd, err := p.commandConfig()
	if err != nil {
		return "", err
	}
	return p.executeCommand(cmd)
}

func (p *ComposeProject) prepareCommand(command string, args []string) (*exec.Cmd, error) {
	// Get cached compose files with processed volume paths
	cachedFiles, err := p.getCachedComposeFiles(p.Project)
	if err != nil {
		slog.Error("Service operation failed",
			"layer", "docker_compose",
			"operation", "prepare_command",
			"project_name", p.Name,
			"error", err)
		return nil, fmt.Errorf("failed to get cached compose files: %w", err)
	}

	// Build docker compose command
	commandArgs := []string{
		"--host", p.Config.DockerHost,
		"compose",
		"--progress", "plain",
		"--project-name", p.Name,
		"--project-directory", p.WorkingDir, // Set working directory for build context
	}

	// Add cached compose files to the command
	for _, cachedFile := range cachedFiles {
		commandArgs = append(commandArgs, "--file", cachedFile)
	}

	// Add the specific command and its arguments
	commandArgs = append(commandArgs, command)
	commandArgs = append(commandArgs, args...)

	slog.Debug("Executing Docker Compose command",
		"command", p.Config.DockerCommand,
		"args", commandArgs,
		"project_name", p.Name,
		"cached_files", cachedFiles)

	// Create command
	cmd := exec.Command(p.Config.DockerCommand, commandArgs...)

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

func (p *ComposeProject) commandUp() (*exec.Cmd, error) {
	return p.prepareCommand("up", []string{"--detach", "--wait", "--quiet-pull", "--no-color", "--remove-orphans"})
}

func (p *ComposeProject) commandDown() (*exec.Cmd, error) {
	return p.prepareCommand("down", []string{"--remove-orphans"})
}

func (p *ComposeProject) commandLogs() (*exec.Cmd, error) {
	return p.prepareCommand("logs", []string{"--follow"})
}

func (p *ComposeProject) commandConfig() (*exec.Cmd, error) {
	return p.prepareCommand("config", []string{})
}

func (p *ComposeProject) commandPs() (*exec.Cmd, error) {
	return p.prepareCommand("ps", []string{"--format", "json"})
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
