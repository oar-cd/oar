package start

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

func TestNewCmdStart(t *testing.T) {
	cmd := NewCmdStart()

	// Test command configuration
	assert.Equal(t, "start", cmd.Use)
	assert.Equal(t, "Start the Oar web service", cmd.Short)
	assert.Contains(t, cmd.Long, "Start the Oar web service container")
	assert.Contains(t, cmd.Long, "docker compose up -d")

	// Test that RunE is set
	assert.NotNil(t, cmd.RunE)

	// Test command has no subcommands by default
	assert.Empty(t, cmd.Commands())
}

func TestNewCmdStart_CommandStructure(t *testing.T) {
	cmd := NewCmdStart()

	// Verify it's a runnable command
	assert.True(t, cmd.Runnable())

	// Verify it accepts args (no specific args requirements)
	assert.Nil(t, cmd.Args)

	// Verify the command can be found by name
	assert.Equal(t, "start", cmd.Name())
}

// Test that the command is configured with proper run function
func TestRunStartConfiguration(t *testing.T) {
	cmd := NewCmdStart()
	require.NotNil(t, cmd.RunE)

	// Verify the RunE function is properly set (not nil)
	// We don't call it to avoid runtime dependencies
	assert.NotNil(t, cmd.RunE)
}

func TestRunStart_Success(t *testing.T) {
	// Create a mock compose project
	mockCompose := &mocks.MockComposeProject{}
	mockCompose.On("UpPiping").Return(nil)

	// Set up the mock function
	utils.SetCreateOarServiceComposeProjectForTesting(
		func(cmd *cobra.Command) (services.ComposeProjectInterface, error) {
			return mockCompose, nil
		},
	)
	defer utils.ResetCreateOarServiceComposeProjectForTesting()

	// Create command and capture output
	cmd := NewCmdStart()
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
	assert.Contains(t, outputStr, "Starting Oar service...")
	assert.Contains(t, outputStr, "Oar service started successfully")
}

func TestRunStart_CreateComposeProjectError(t *testing.T) {
	// Set up the mock function to return an error
	expectedError := errors.New("compose project creation failed")
	utils.SetCreateOarServiceComposeProjectForTesting(
		func(cmd *cobra.Command) (services.ComposeProjectInterface, error) {
			return nil, expectedError
		},
	)
	defer utils.ResetCreateOarServiceComposeProjectForTesting()

	// Create command
	cmd := NewCmdStart()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	// Execute the command
	err := cmd.RunE(cmd, []string{})

	// Verify error is returned
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
}

func TestRunStart_UpPipingError(t *testing.T) {
	// Create a mock compose project that fails on UpPiping
	mockCompose := &mocks.MockComposeProject{}
	upError := errors.New("docker compose up failed")
	mockCompose.On("UpPiping").Return(upError)

	// Set up the mock function
	utils.SetCreateOarServiceComposeProjectForTesting(
		func(cmd *cobra.Command) (services.ComposeProjectInterface, error) {
			return mockCompose, nil
		},
	)
	defer utils.ResetCreateOarServiceComposeProjectForTesting()

	// Create command and capture output
	cmd := NewCmdStart()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	// Execute the command
	err := cmd.RunE(cmd, []string{})

	// Verify error is returned with proper wrapping
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to start Oar service")
	assert.Contains(t, err.Error(), "docker compose up failed")
	mockCompose.AssertExpectations(t)

	// Check that starting message was shown but success message was not
	outputStr := output.String()
	assert.Contains(t, outputStr, "Starting Oar service...")
	assert.NotContains(t, outputStr, "Oar service started successfully")
}
