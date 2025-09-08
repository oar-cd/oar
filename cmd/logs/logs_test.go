package logs

import (
	"testing"

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
