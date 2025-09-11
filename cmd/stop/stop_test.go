package stop

import (
	"bytes"
	"errors"
	"github.com/oar-cd/oar/services"
	"testing"

	"github.com/oar-cd/oar/cmd/utils"
	"github.com/oar-cd/oar/testing/mocks"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCmdStop(t *testing.T) {
	cmd := NewCmdStop()

	// Test command configuration
	assert.Equal(t, "stop", cmd.Use)
	assert.Equal(t, "Stop the Oar web service", cmd.Short)
	assert.Contains(t, cmd.Long, "Stop the Oar web service container")
	assert.Contains(t, cmd.Long, "docker compose down")

	// Test that RunE is set
	assert.NotNil(t, cmd.RunE)

	// Test command has no subcommands by default
	assert.Empty(t, cmd.Commands())
}

func TestNewCmdStop_CommandStructure(t *testing.T) {
	cmd := NewCmdStop()

	// Verify it's a runnable command
	assert.True(t, cmd.Runnable())

	// Verify it accepts args (no specific args requirements)
	assert.Nil(t, cmd.Args)

	// Verify the command can be found by name
	assert.Equal(t, "stop", cmd.Name())
}

// Test that the command is configured with proper run function
func TestRunStopConfiguration(t *testing.T) {
	cmd := NewCmdStop()
	require.NotNil(t, cmd.RunE)

	// Verify the RunE function is properly set (not nil)
	// We don't call it to avoid runtime dependencies
	assert.NotNil(t, cmd.RunE)
}

func TestRunStop_Success(t *testing.T) {
	// Create a mock compose project
	mockCompose := &mocks.MockComposeProject{}
	mockCompose.On("DownPiping").Return(nil)

	// Set up the mock function
	utils.SetCreateOarServiceComposeProjectForTesting(
		func(cmd *cobra.Command) (services.ComposeProjectInterface, error) {
			return mockCompose, nil
		},
	)
	defer utils.ResetCreateOarServiceComposeProjectForTesting()

	// Create command and capture output
	cmd := NewCmdStop()
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
	assert.Contains(t, outputStr, "Stopping Oar service...")
	assert.Contains(t, outputStr, "Oar service stopped successfully")
}

func TestRunStop_CreateComposeProjectError(t *testing.T) {
	// Set up the mock function to return an error
	expectedError := errors.New("compose project creation failed")
	utils.SetCreateOarServiceComposeProjectForTesting(
		func(cmd *cobra.Command) (services.ComposeProjectInterface, error) {
			return nil, expectedError
		},
	)
	defer utils.ResetCreateOarServiceComposeProjectForTesting()

	// Create command
	cmd := NewCmdStop()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	// Execute the command
	err := cmd.RunE(cmd, []string{})

	// Verify error is returned
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
}

func TestRunStop_DownPipingError(t *testing.T) {
	// Create a mock compose project that fails on DownPiping
	mockCompose := &mocks.MockComposeProject{}
	downError := errors.New("docker compose down failed")
	mockCompose.On("DownPiping").Return(downError)

	// Set up the mock function
	utils.SetCreateOarServiceComposeProjectForTesting(
		func(cmd *cobra.Command) (services.ComposeProjectInterface, error) {
			return mockCompose, nil
		},
	)
	defer utils.ResetCreateOarServiceComposeProjectForTesting()

	// Create command and capture output
	cmd := NewCmdStop()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	// Execute the command
	err := cmd.RunE(cmd, []string{})

	// Verify error is returned with proper wrapping
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to stop Oar service")
	assert.Contains(t, err.Error(), "docker compose down failed")
	mockCompose.AssertExpectations(t)

	// Check that stopping message was shown but success message was not
	outputStr := output.String()
	assert.Contains(t, outputStr, "Stopping Oar service...")
	assert.NotContains(t, outputStr, "Oar service stopped successfully")
}
