package stop

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCmdStop(t *testing.T) {
	cmd := NewCmdStop()

	// Test command configuration
	assert.Equal(t, "stop", cmd.Use)
	assert.Contains(t, cmd.Short, "Stop")
	assert.Contains(t, cmd.Short, "Oar web service")

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
