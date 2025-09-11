package logs

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

func TestNewCmdLogs(t *testing.T) {
	cmd := NewCmdLogs()

	// Test command configuration
	assert.Equal(t, "logs", cmd.Use)
	assert.Equal(t, "View logs from the Oar web service", cmd.Short)
	assert.Contains(t, cmd.Long, "Display logs from the running Oar web service container")
	assert.Contains(t, cmd.Long, "docker compose logs --follow")
	assert.Contains(t, cmd.Long, "Press Ctrl+C to stop")

	// Test that RunE is set
	assert.NotNil(t, cmd.RunE)

	// Test command has no subcommands by default
	assert.Empty(t, cmd.Commands())
}

func TestNewCmdLogs_CommandStructure(t *testing.T) {
	cmd := NewCmdLogs()

	// Verify it's a runnable command
	assert.True(t, cmd.Runnable())

	// Verify it accepts args (no specific args requirements)
	assert.Nil(t, cmd.Args)

	// Verify the command can be found by name
	assert.Equal(t, "logs", cmd.Name())
}

// Test that the command is configured with proper run function
func TestRunLogsConfiguration(t *testing.T) {
	cmd := NewCmdLogs()
	require.NotNil(t, cmd.RunE)

	// Verify the RunE function is properly set (not nil)
	// We don't call it to avoid runtime dependencies
	assert.NotNil(t, cmd.RunE)
}

func TestRunLogs_Success(t *testing.T) {
	// Create a mock compose project
	mockCompose := &mocks.MockComposeProject{}
	mockCompose.On("LogsPiping").Return(nil)

	// Set up the mock function
	utils.SetCreateOarServiceComposeProjectForTesting(
		func(cmd *cobra.Command) (services.ComposeProjectInterface, error) {
			return mockCompose, nil
		},
	)
	defer utils.ResetCreateOarServiceComposeProjectForTesting()

	// Create command and capture output
	cmd := NewCmdLogs()
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
	assert.Contains(t, outputStr, "Streaming logs from Oar service...")
	assert.Contains(t, outputStr, "Press Ctrl+C to stop")
}

func TestRunLogs_CreateComposeProjectError(t *testing.T) {
	// Set up the mock function to return an error
	expectedError := errors.New("compose project creation failed")
	utils.SetCreateOarServiceComposeProjectForTesting(
		func(cmd *cobra.Command) (services.ComposeProjectInterface, error) {
			return nil, expectedError
		},
	)
	defer utils.ResetCreateOarServiceComposeProjectForTesting()

	// Create command
	cmd := NewCmdLogs()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	// Execute the command
	err := cmd.RunE(cmd, []string{})

	// Verify error is returned
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
}

func TestRunLogs_LogsPipingError(t *testing.T) {
	// Create a mock compose project that fails on LogsPiping
	mockCompose := &mocks.MockComposeProject{}
	logsError := errors.New("docker compose logs failed")
	mockCompose.On("LogsPiping").Return(logsError)

	// Set up the mock function
	utils.SetCreateOarServiceComposeProjectForTesting(
		func(cmd *cobra.Command) (services.ComposeProjectInterface, error) {
			return mockCompose, nil
		},
	)
	defer utils.ResetCreateOarServiceComposeProjectForTesting()

	// Create command and capture output
	cmd := NewCmdLogs()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	// Execute the command
	err := cmd.RunE(cmd, []string{})

	// Verify error is returned with proper wrapping
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get logs")
	assert.Contains(t, err.Error(), "docker compose logs failed")
	mockCompose.AssertExpectations(t)

	// Check that starting messages were shown
	outputStr := output.String()
	assert.Contains(t, outputStr, "Streaming logs from Oar service...")
	assert.Contains(t, outputStr, "Press Ctrl+C to stop")
}
