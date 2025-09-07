package start

import (
	"testing"

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
