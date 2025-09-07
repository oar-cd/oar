package status

import (
	"testing"

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
