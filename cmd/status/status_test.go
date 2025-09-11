package status

import (
	"bytes"
	"errors"
	"github.com/ch00k/oar/services"
	"testing"

	"github.com/ch00k/oar/cmd/utils"
	"github.com/ch00k/oar/testing/mocks"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCmdStatus(t *testing.T) {
	cmd := NewCmdStatus()

	// Test command configuration
	assert.Equal(t, "status", cmd.Use)
	assert.Equal(t, "Show the status of the Oar web service", cmd.Short)
	assert.Contains(t, cmd.Long, "Show the status of the Oar web service container")
	assert.Contains(t, cmd.Long, "status information")

	// Test that RunE is set
	assert.NotNil(t, cmd.RunE)

	// Test command has no subcommands by default
	assert.Empty(t, cmd.Commands())
}

func TestNewCmdStatus_CommandStructure(t *testing.T) {
	cmd := NewCmdStatus()

	// Verify it's a runnable command
	assert.True(t, cmd.Runnable())

	// Verify it accepts args (no specific args requirements)
	assert.Nil(t, cmd.Args)

	// Verify the command can be found by name
	assert.Equal(t, "status", cmd.Name())
}

// Test that the command is configured with proper run function
func TestRunStatusConfiguration(t *testing.T) {
	cmd := NewCmdStatus()
	require.NotNil(t, cmd.RunE)

	// Verify the RunE function is properly set (not nil)
	// We don't call it to avoid runtime dependencies
	assert.NotNil(t, cmd.RunE)
}

func TestRunStatus_Success_Running(t *testing.T) {
	// Create a mock compose project with running status
	mockCompose := &mocks.MockComposeProject{}
	status := &services.ComposeStatus{
		Status: "running",
		Uptime: "2 hours",
		Containers: []services.ContainerInfo{
			{Service: "oar", State: "running", Status: "Up 2 hours"},
			{Service: "db", State: "running", Status: "Up 2 hours"},
		},
	}
	mockCompose.On("Status").Return(status, nil)

	// Set up the mock function
	utils.SetCreateOarServiceComposeProjectForTesting(
		func(cmd *cobra.Command) (services.ComposeProjectInterface, error) {
			return mockCompose, nil
		},
	)
	defer utils.ResetCreateOarServiceComposeProjectForTesting()

	// Create command and capture output
	cmd := NewCmdStatus()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	// Execute the command
	err := cmd.RunE(cmd, []string{})

	// Verify success
	assert.NoError(t, err)
	mockCompose.AssertExpectations(t)

	// Check output contains expected messages
	outputStr := output.String()
	assert.Contains(t, outputStr, "Status: running")
	assert.Contains(t, outputStr, "Uptime: 2 hours")
	assert.Contains(t, outputStr, "Containers:")
	assert.Contains(t, outputStr, "[OK] oar: Up 2 hours")
	assert.Contains(t, outputStr, "[OK] db: Up 2 hours")
}

func TestRunStatus_Success_Stopped(t *testing.T) {
	// Create a mock compose project with stopped status
	mockCompose := &mocks.MockComposeProject{}
	status := &services.ComposeStatus{
		Status: "stopped",
		Uptime: "",
		Containers: []services.ContainerInfo{
			{Service: "oar", State: "exited", Status: "Exited (0) 1 hour ago"},
		},
	}
	mockCompose.On("Status").Return(status, nil)

	// Set up the mock function
	utils.SetCreateOarServiceComposeProjectForTesting(
		func(cmd *cobra.Command) (services.ComposeProjectInterface, error) {
			return mockCompose, nil
		},
	)
	defer utils.ResetCreateOarServiceComposeProjectForTesting()

	// Create command and capture output
	cmd := NewCmdStatus()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	// Execute the command
	err := cmd.RunE(cmd, []string{})

	// Verify success
	assert.NoError(t, err)
	mockCompose.AssertExpectations(t)

	// Check output contains expected messages
	outputStr := output.String()
	assert.Contains(t, outputStr, "Status: stopped")
	assert.NotContains(t, outputStr, "Uptime:") // Should not show uptime when stopped
	assert.Contains(t, outputStr, "Containers:")
	assert.Contains(t, outputStr, "[ERROR] oar: Exited (0) 1 hour ago")
}

func TestRunStatus_CreateComposeProjectError(t *testing.T) {
	// Set up the mock function to return an error
	expectedError := errors.New("compose project creation failed")
	utils.SetCreateOarServiceComposeProjectForTesting(
		func(cmd *cobra.Command) (services.ComposeProjectInterface, error) {
			return nil, expectedError
		},
	)
	defer utils.ResetCreateOarServiceComposeProjectForTesting()

	// Create command
	cmd := NewCmdStatus()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	// Execute the command
	err := cmd.RunE(cmd, []string{})

	// Verify error is returned
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
}

func TestRunStatus_StatusError(t *testing.T) {
	// Create a mock compose project that fails on Status
	mockCompose := &mocks.MockComposeProject{}
	statusError := errors.New("docker compose ps failed")
	mockCompose.On("Status").Return((*services.ComposeStatus)(nil), statusError)

	// Set up the mock function
	utils.SetCreateOarServiceComposeProjectForTesting(
		func(cmd *cobra.Command) (services.ComposeProjectInterface, error) {
			return mockCompose, nil
		},
	)
	defer utils.ResetCreateOarServiceComposeProjectForTesting()

	// Create command and capture output
	cmd := NewCmdStatus()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	// Execute the command
	err := cmd.RunE(cmd, []string{})

	// Verify error is returned with proper wrapping
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get Oar service status")
	assert.Contains(t, err.Error(), "docker compose ps failed")
	mockCompose.AssertExpectations(t)
}
