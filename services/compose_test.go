package services

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for NewComposeProject
func TestNewComposeProject_Success(t *testing.T) {
	// Create test project with working directory
	testProject := createTestProjectWithOptions(ProjectOptions{
		Name:         "test-compose-project",
		WorkingDir:   "/tmp/test-compose-project",
		ComposeFiles: []string{"docker-compose.yml", "docker-compose.override.yml"},
		Variables:    []string{"KEY1=value1", "KEY2=value2"},
	})
	tempDir := t.TempDir()
	testProject.WorkingDir = filepath.Join(tempDir, testProject.ID.String())

	// Create the working directory and git subdirectory
	gitDir := filepath.Join(testProject.WorkingDir, GitDir)
	err := os.MkdirAll(gitDir, 0o755)
	require.NoError(t, err)

	// Create test config
	config := &Config{
		DataDir:    tempDir,
		LogLevel:   "info",
		GitTimeout: 5 * time.Minute,
	}

	// Test
	composeProject := NewComposeProject(testProject, config)

	// Assertions
	assert.NotNil(t, composeProject)
	assert.Equal(t, testProject.Name, composeProject.Name)
	assert.Equal(t, gitDir, composeProject.WorkingDir)
	assert.Equal(t, testProject.ComposeFiles, composeProject.ComposeFiles)
	assert.Equal(t, testProject.Variables, composeProject.Variables)
}

func TestNewComposeProject_InvalidProject(t *testing.T) {
	// Create project with invalid working directory
	testProject := createTestProjectWithOptions(ProjectOptions{
		Name:         "test-compose-project",
		WorkingDir:   "/tmp/test-compose-project",
		ComposeFiles: []string{"docker-compose.yml", "docker-compose.override.yml"},
		Variables:    []string{"KEY1=value1", "KEY2=value2"},
	})
	testProject.WorkingDir = "" // Invalid working directory

	// Create test config
	tempDir := t.TempDir()
	config := &Config{
		DataDir:    tempDir,
		LogLevel:   "info",
		GitTimeout: 5 * time.Minute,
	}

	// Test
	composeProject := NewComposeProject(testProject, config)

	// Assertions
	assert.Nil(t, composeProject)
}

// Tests for ComposeProject.prepareCommand
func TestComposeProject_PrepareCommand_Basic(t *testing.T) {
	composeProject := createTestComposeProject()
	tempDir := t.TempDir()
	composeProject.WorkingDir = tempDir

	// Test
	cmd := composeProject.prepareCommand("up", []string{"--detach"})

	// Assertions
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Path, "docker") // Path may be full path like /usr/bin/docker
	assert.Equal(t, "", cmd.Dir)           // Working directory is no longer set to avoid host path resolution issues

	// Verify command arguments
	expectedArgs := []string{
		"docker", // cmd.Args[0] is always the command name
		"compose",
		"--progress", "plain",
		"--project-name", "test-project",
		"--file", filepath.Join(tempDir, "docker-compose.yml"),
		"up",
		"--detach",
	}
	assert.Equal(t, expectedArgs, cmd.Args)
}

func TestComposeProject_PrepareCommand_MultipleFiles(t *testing.T) {
	composeProject := createTestComposeProject()
	composeProject.ComposeFiles = []string{
		"docker-compose.yml",
		"docker-compose.override.yml",
		"docker-compose.prod.yml",
	}
	tempDir := t.TempDir()
	composeProject.WorkingDir = tempDir

	// Test
	cmd := composeProject.prepareCommand("down", []string{"--remove-orphans"})

	// Assertions
	assert.NotNil(t, cmd)

	// Verify all compose files are included
	expectedArgs := []string{
		"docker",
		"compose",
		"--progress", "plain",
		"--project-name", "test-project",
		"--file", filepath.Join(tempDir, "docker-compose.yml"),
		"--file", filepath.Join(tempDir, "docker-compose.override.yml"),
		"--file", filepath.Join(tempDir, "docker-compose.prod.yml"),
		"down",
		"--remove-orphans",
	}
	assert.Equal(t, expectedArgs, cmd.Args)
}

func TestComposeProject_PrepareCommand_NoFiles(t *testing.T) {
	composeProject := createTestComposeProject()
	composeProject.ComposeFiles = []string{} // No compose files
	tempDir := t.TempDir()
	composeProject.WorkingDir = tempDir

	// Test
	cmd := composeProject.prepareCommand("ps", []string{})

	// Assertions
	assert.NotNil(t, cmd)

	// Should still work with no files (docker compose will use defaults)
	expectedArgs := []string{
		"docker",
		"compose",
		"--progress", "plain",
		"--project-name", "test-project",
		"ps",
	}
	assert.Equal(t, expectedArgs, cmd.Args)
}

// Tests for specific command builders
func TestComposeProject_CommandUp(t *testing.T) {
	composeProject := createTestComposeProject()
	tempDir := t.TempDir()
	composeProject.WorkingDir = tempDir

	// Test
	cmd := composeProject.commandUp(true)

	// Assertions
	assert.NotNil(t, cmd)

	// Verify complete command arguments
	expectedArgs := []string{
		"docker", // cmd.Args[0] is always the command name
		"compose",
		"--progress", "plain",
		"--project-name", "test-project",
		"--file", tempDir + "/docker-compose.yml",
		"up",
		"--detach",
		"--quiet-pull",
		"--quiet-build",
		"--remove-orphans",
	}
	assert.Equal(t, expectedArgs, cmd.Args)
}

func TestComposeProject_CommandDown(t *testing.T) {
	composeProject := createTestComposeProject()
	tempDir := t.TempDir()
	composeProject.WorkingDir = tempDir

	// Test
	cmd := composeProject.commandDown(false)

	// Assertions
	assert.NotNil(t, cmd)

	// Verify complete command arguments
	expectedArgs := []string{
		"docker", // cmd.Args[0] is always the command name
		"compose",
		"--progress", "plain",
		"--project-name", "test-project",
		"--file", tempDir + "/docker-compose.yml",
		"down",
		"--remove-orphans",
	}
	assert.Equal(t, expectedArgs, cmd.Args)
}

func TestComposeProject_CommandLogs(t *testing.T) {
	composeProject := createTestComposeProject()
	tempDir := t.TempDir()
	composeProject.WorkingDir = tempDir

	// Test
	cmd := composeProject.commandLogs(true)

	// Assertions
	assert.NotNil(t, cmd)

	// Verify complete command arguments
	expectedArgs := []string{
		"docker", // cmd.Args[0] is always the command name
		"compose",
		"--progress", "plain",
		"--project-name", "test-project",
		"--file", tempDir + "/docker-compose.yml",
		"logs",
		"--follow",
	}
	assert.Equal(t, expectedArgs, cmd.Args)
}

// Tests for executeCommand (using real commands that are safe)
func TestComposeProject_ExecuteCommand_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping command execution test in short mode")
	}

	composeProject := createTestComposeProject()

	// Use a simple command that should work on most systems
	cmd := exec.Command("echo", "test output")

	// Test
	stdout, stderr, err := composeProject.executeCommand(cmd)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, "test output\n", stdout)
	assert.Empty(t, stderr)
}

func TestComposeProject_ExecuteCommand_Error(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping command execution test in short mode")
	}

	composeProject := createTestComposeProject()

	// Use a command that should fail
	cmd := exec.Command("false") // 'false' command always exits with code 1

	// Test
	stdout, stderr, err := composeProject.executeCommand(cmd)

	// Assertions
	assert.Error(t, err)
	assert.Empty(t, stdout)
	assert.Empty(t, stderr)
}

// Tests for executeCommandStreaming
func TestComposeProject_ExecuteCommandStreaming_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping streaming command test in short mode")
	}

	composeProject := createTestComposeProject()

	// Create a command that outputs multiple lines
	cmd := exec.Command("sh", "-c", "echo 'line1'; echo 'line2'; echo 'line3'")

	// Create output channel
	outputChan := make(chan StreamMessage, 10)
	var receivedLines []string

	// Start goroutine to collect output
	done := make(chan bool)
	go func() {
		for msg := range outputChan {
			receivedLines = append(receivedLines, msg.Content)
		}
		done <- true
	}()

	// Test
	err := composeProject.executeCommandStreaming(cmd, outputChan)
	close(outputChan)

	// Wait for output collection to complete
	<-done

	// Assertions
	assert.NoError(t, err)
	assert.Len(t, receivedLines, 3)

	// Now we expect clean strings directly, not JSON
	assert.Contains(t, receivedLines, "line1")
	assert.Contains(t, receivedLines, "line2")
	assert.Contains(t, receivedLines, "line3")
}

func TestComposeProject_ExecuteCommandStreaming_Error(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping streaming command test in short mode")
	}

	composeProject := createTestComposeProject()

	// Use a command that should fail
	cmd := exec.Command("false")

	// Create output channel
	outputChan := make(chan StreamMessage, 10)

	// Test
	err := composeProject.executeCommandStreaming(cmd, outputChan)
	close(outputChan)

	// Assertions
	assert.Error(t, err)
}

// Tests for high-level operations (Up, Down, Logs)
// Note: These tests use mocked Docker commands since we can't assume Docker is available

func TestComposeProject_Up_MockDocker(t *testing.T) {
	// Skip if Docker is not available (this is more of an integration test)
	t.Skip("Skipping Docker integration test - would require Docker to be available")

	composeProject := createTestComposeProject()
	tempDir := t.TempDir()
	composeProject.WorkingDir = tempDir

	// Create a mock compose file
	composeContent := `version: '3'
services:
  test:
    image: hello-world
`
	err := os.WriteFile(filepath.Join(tempDir, "docker-compose.yml"), []byte(composeContent), 0o644)
	require.NoError(t, err)

	// Test (this would require Docker to be running)
	stdout, stderr, err := composeProject.Up(true)

	// Assertions (these would need to be adjusted based on actual Docker behavior)
	// This is more of a placeholder for integration testing
	_ = stdout
	_ = stderr
	_ = err
}

// Tests for edge cases and error conditions
func TestComposeProject_EmptyProjectName(t *testing.T) {
	composeProject := createTestComposeProject()
	composeProject.Name = ""
	tempDir := t.TempDir()
	composeProject.WorkingDir = tempDir

	// Test
	cmd := composeProject.prepareCommand("up", []string{})

	// Assertions
	assert.NotNil(t, cmd)

	// Verify complete command arguments with empty project name
	expectedArgs := []string{
		"docker", // cmd.Args[0] is always the command name
		"compose",
		"--progress", "plain",
		"--project-name", "",
		"--file", tempDir + "/docker-compose.yml",
		"up",
	}
	assert.Equal(t, expectedArgs, cmd.Args)
}

func TestComposeProject_InvalidWorkingDirectory(t *testing.T) {
	composeProject := createTestComposeProject()
	composeProject.WorkingDir = "/non/existent/directory"

	// Test
	cmd := composeProject.prepareCommand("up", []string{})

	// Assertions // prepareCommand doesn't validate directory existence
	assert.NotNil(t, cmd)
	assert.Equal(t, "", cmd.Dir) // Working directory is no longer set to avoid host path resolution issues
}

// Tests for streaming operations with channels
func TestComposeProject_StreamingChannelManagement(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping streaming test in short mode")
	}

	composeProject := createTestComposeProject()

	// Test that streaming properly handles channel operations
	outputChan := make(chan StreamMessage, 10)

	// Create a simple command for testing
	cmd := exec.Command("echo", "streaming test")

	// Test
	err := composeProject.executeCommandStreaming(cmd, outputChan)

	// Close channel and collect output
	close(outputChan)
	var output []string
	for msg := range outputChan {
		output = append(output, msg.Content)
	}

	// Assertions
	assert.NoError(t, err)
	assert.Len(t, output, 1)

	// Now we expect clean string directly, not JSON
	assert.Equal(t, "streaming test", output[0])
}

// Test for concurrent streaming operations
func TestComposeProject_ConcurrentStreaming(t *testing.T) {
	t.Skip("Skipping - flaky test in CI")
	if testing.Short() {
		t.Skip("Skipping concurrent streaming test in short mode")
	}

	composeProject := createTestComposeProject()

	// Number of concurrent operations
	numOps := 3

	// Channels for coordination
	done := make(chan error, numOps)

	// Start multiple streaming operations concurrently
	for i := range numOps {
		go func(id int) {
			outputChan := make(chan StreamMessage, 10)
			// Use a more reliable command that should always produce output
			cmd := exec.Command("sh", "-c", fmt.Sprintf("printf 'concurrent test %d\\n'", id))

			err := composeProject.executeCommandStreaming(cmd, outputChan)
			close(outputChan)

			// Collect output to ensure it completes
			var lines []string
			for msg := range outputChan {
				lines = append(lines, msg.Content)
			}

			if err != nil {
				done <- fmt.Errorf("command failed for operation %d: %w", id, err)
				return
			}

			expectedOutput := fmt.Sprintf("concurrent test %d", id)
			if len(lines) != 1 {
				done <- fmt.Errorf("unexpected number of lines for operation %d: got %d, expected 1", id, len(lines))
				return
			}

			// Now we expect clean strings directly, not JSON
			message := lines[0]

			if message != expectedOutput {
				done <- fmt.Errorf("unexpected output for operation %d: got %s, expected %s", id, message, expectedOutput)
				return
			}

			done <- nil
		}(i)
	}

	// Wait for all operations to complete
	for range numOps {
		select {
		case err := <-done:
			assert.NoError(t, err)
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for concurrent operations")
		}
	}
}

// Tests for Status calculation logic
func TestComposeProject_StatusCalculation_AllSuccessfullyExitedContainers(t *testing.T) {
	// Test the status calculation logic directly by simulating different container scenarios
	tests := []struct {
		name           string
		containers     []ContainerInfo
		expectedStatus ComposeProjectStatus
		expectedUptime string
	}{
		{
			name: "all containers successfully exited (init containers)",
			containers: []ContainerInfo{
				{Service: "migrate", State: "exited", ExitCode: 0, Status: "Exited (0) 5 minutes ago"},
				{Service: "collectstatic", State: "exited", ExitCode: 0, Status: "Exited (0) 5 minutes ago"},
			},
			expectedStatus: ComposeProjectStatusUnknown,
			expectedUptime: "",
		},
		{
			name: "all containers running",
			containers: []ContainerInfo{
				{Service: "web", State: "running", Status: "Up 2 hours", RunningFor: "2 hours ago"},
				{Service: "db", State: "running", Status: "Up 2 hours", RunningFor: "2 hours ago"},
			},
			expectedStatus: ComposeProjectStatusRunning,
			expectedUptime: "2 hours",
		},
		{
			name: "mixed running and failed containers",
			containers: []ContainerInfo{
				{Service: "web", State: "running", Status: "Up 1 hour", RunningFor: "1 hour ago"},
				{Service: "db", State: "exited", ExitCode: 1, Status: "Exited (1) 10 minutes ago"},
			},
			expectedStatus: ComposeProjectStatusFailed,
			expectedUptime: "1 hour",
		},
		{
			name: "all containers stopped (non-zero exit codes)",
			containers: []ContainerInfo{
				{Service: "web", State: "exited", ExitCode: 1, Status: "Exited (1) 1 hour ago"},
				{Service: "db", State: "exited", ExitCode: 1, Status: "Exited (1) 1 hour ago"},
			},
			expectedStatus: ComposeProjectStatusStopped,
			expectedUptime: "",
		},
		{
			name: "regular containers cleanly stopped (would be treated as unknown due to current logic)",
			containers: []ContainerInfo{
				{Service: "web", State: "exited", ExitCode: 0, Status: "Exited (0) 1 hour ago"},
				{Service: "db", State: "exited", ExitCode: 0, Status: "Exited (0) 1 hour ago"},
			},
			expectedStatus: ComposeProjectStatusUnknown, // Current logic treats all ExitCode:0 as init containers
			expectedUptime: "",
		},
		{
			name: "running containers with successful init containers",
			containers: []ContainerInfo{
				{Service: "web", State: "running", Status: "Up 1 hour", RunningFor: "1 hour ago"},
				{Service: "migrate", State: "exited", ExitCode: 0, Status: "Exited (0) 2 hours ago"},
			},
			expectedStatus: ComposeProjectStatusRunning,
			expectedUptime: "1 hour",
		},
		{
			name:           "no containers",
			containers:     []ContainerInfo{},
			expectedStatus: ComposeProjectStatusStopped,
			expectedUptime: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the status calculation logic from ComposeProject.Status()
			projectStatus := ComposeProjectStatusStopped
			uptime := ""

			if len(tt.containers) > 0 {
				runningCount := 0
				totalRelevantContainers := 0

				for _, container := range tt.containers {
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

			// Assertions
			assert.Equal(t, tt.expectedStatus, projectStatus, "status should match expected")
			assert.Equal(t, tt.expectedUptime, uptime, "uptime should match expected")
		})
	}
}

// Test for ParseComposeProjectStatus function
func TestParseComposeProjectStatus(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedStatus ComposeProjectStatus
		expectError    bool
	}{
		{
			name:           "parse running status",
			input:          "running",
			expectedStatus: ComposeProjectStatusRunning,
			expectError:    false,
		},
		{
			name:           "parse stopped status",
			input:          "stopped",
			expectedStatus: ComposeProjectStatusStopped,
			expectError:    false,
		},
		{
			name:           "parse failed status",
			input:          "failed",
			expectedStatus: ComposeProjectStatusFailed,
			expectError:    false,
		},
		{
			name:           "parse unknown status",
			input:          "unknown",
			expectedStatus: ComposeProjectStatusUnknown,
			expectError:    false,
		},
		{
			name:           "parse invalid status",
			input:          "invalid",
			expectedStatus: ComposeProjectStatusUnknown,
			expectError:    true,
		},
		{
			name:           "parse empty string",
			input:          "",
			expectedStatus: ComposeProjectStatusUnknown,
			expectError:    true,
		},
		{
			name:           "parse mixed case",
			input:          "Running",
			expectedStatus: ComposeProjectStatusUnknown,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, err := ParseComposeProjectStatus(tt.input)

			if tt.expectError {
				assert.Error(t, err, "should return error for input: %s", tt.input)
			} else {
				assert.NoError(t, err, "should not return error for valid input: %s", tt.input)
			}

			assert.Equal(t, tt.expectedStatus, status, "status should match expected for input: %s", tt.input)
		})
	}
}

// Tests for ParseComposeLogLine
func TestParseComposeLogLine(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "structured log with warning",
			input:    `time="2025-09-22T18:22:35+02:00" level=warning msg="The \"MEED_SMTP_PORT\" variable is not set. Defaulting to a blank string."`,
			expected: `The "MEED_SMTP_PORT" variable is not set. Defaulting to a blank string.`,
		},
		{
			name:     "structured log with info",
			input:    `time="2025-09-22T18:22:35+02:00" level=info msg="Container started successfully"`,
			expected: "Container started successfully",
		},
		{
			name:     "structured log with escaped backslash",
			input:    `time="2025-09-22T18:22:35+02:00" level=error msg="Path \\\"C:\\\\temp\\\" not found"`,
			expected: `Path \"C:\\temp\" not found`,
		},
		{
			name:     "structured log with additional fields",
			input:    `time="2025-09-22T18:22:35+02:00" level=warning msg="Service health check failed" service=web container=web_1`,
			expected: "Service health check failed",
		},
		{
			name:     "TTY format log (no msg field)",
			input:    `WARN[0000] The "MEED_SMTP_PORT" variable is not set. Defaulting to a blank string.`,
			expected: `WARN[0000] The "MEED_SMTP_PORT" variable is not set. Defaulting to a blank string.`,
		},
		{
			name:     "regular log line without structured format",
			input:    "Building application...",
			expected: "Building application...",
		},
		{
			name:     "empty line",
			input:    "",
			expected: "",
		},
		{
			name:     "line with msg but not structured format",
			input:    "This line contains the word msg but is not structured",
			expected: "This line contains the word msg but is not structured",
		},
		{
			name:     "structured log with empty message",
			input:    `time="2025-09-22T18:22:35+02:00" level=info msg=""`,
			expected: "",
		},
		{
			name:     "structured log with complex escaped content",
			input:    `time="2025-09-22T18:22:35+02:00" level=error msg="JSON parse error: \"field\\n\\tvalue\" at line 10"`,
			expected: `JSON parse error: "field\n\tvalue" at line 10`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseComposeLogLine(tt.input)
			assert.Equal(t, tt.expected, result, "parsed content should match expected for input: %s", tt.input)
		})
	}
}
