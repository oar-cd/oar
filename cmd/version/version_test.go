package version

import (
	"testing"

	"github.com/oar-cd/oar/app"
	"github.com/stretchr/testify/assert"
)

func TestNewCmdVersion(t *testing.T) {
	cmd := NewCmdVersion()

	// Test command configuration
	assert.Equal(t, "version", cmd.Use)
	assert.Equal(t, "Show version information", cmd.Short)
	assert.Contains(t, cmd.Long, "Display version information for Oar")

	// Test that RunE is set
	assert.NotNil(t, cmd.RunE)

	// Test command has no flags
	assert.Empty(t, cmd.Flags().FlagUsages())

	// Test command has no subcommands by default
	assert.Empty(t, cmd.Commands())

	// Verify it's a runnable command
	assert.True(t, cmd.Runnable())

	// Verify the command can be found by name
	assert.Equal(t, "version", cmd.Name())
}

func TestVersionVariable(t *testing.T) {
	// Test that Version has a default value
	assert.NotEmpty(t, app.Version)
	assert.Equal(t, "dev", app.Version) // Default build-time value
}

// Test that we can call runVersion without it panicking
// Note: This will print to stdout but won't cause test failure
func TestRunVersionExecutes(t *testing.T) {
	err := runVersion()
	assert.NoError(t, err)
}
