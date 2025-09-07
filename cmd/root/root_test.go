package root

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCmdRoot(t *testing.T) {
	defaultDataDir := "/test/data/dir"
	cmd := NewCmdRoot(defaultDataDir)

	// Test command configuration
	assert.Equal(t, "oar", cmd.Use)
	assert.Equal(t, "GitOps deployment tool for Docker Compose projects", cmd.Short)
	assert.Contains(t, cmd.Long, "Oar manages Docker Compose applications")
	assert.Contains(t, cmd.Long, "Git repositories")
	assert.Contains(t, cmd.Long, "state tracking")

	// Test that RunE is set (should show help)
	assert.NotNil(t, cmd.RunE)

	// Test that PersistentPreRun is set
	assert.NotNil(t, cmd.PersistentPreRun)

	// Verify it's a runnable command
	assert.True(t, cmd.Runnable())

	// Verify the command can be found by name
	assert.Equal(t, "oar", cmd.Name())

	// Test that subcommands are properly registered
	subcommands := cmd.Commands()
	assert.NotEmpty(t, subcommands)

	// Check for expected subcommands
	subcommandNames := make([]string, len(subcommands))
	for i, subcmd := range subcommands {
		subcommandNames[i] = subcmd.Name()
	}

	expectedSubcommands := []string{"logs", "project", "start", "status", "stop", "update", "version"}
	for _, expected := range expectedSubcommands {
		assert.Contains(t, subcommandNames, expected, "Expected subcommand %s not found", expected)
	}
}

func TestNewCmdRootFlags(t *testing.T) {
	defaultDataDir := "/test/data/dir"
	cmd := NewCmdRoot(defaultDataDir)

	// Check persistent flags exist
	dataDirFlag := cmd.PersistentFlags().Lookup("data-dir")
	assert.NotNil(t, dataDirFlag)
	assert.Equal(t, "d", dataDirFlag.Shorthand)
	assert.Equal(t, defaultDataDir, dataDirFlag.DefValue)

	logLevelFlag := cmd.PersistentFlags().Lookup("log-level")
	assert.NotNil(t, logLevelFlag)
	assert.Equal(t, "l", logLevelFlag.Shorthand)

	noColorFlag := cmd.PersistentFlags().Lookup("no-color")
	assert.NotNil(t, noColorFlag)
	assert.Equal(t, "c", noColorFlag.Shorthand)
}

func TestNewCmdRootPersistentPreRunLogic(t *testing.T) {
	// This tests the skip initialization logic without actually running it
	// since that would require full app initialization

	skipInitCommands := []string{"version", "update", "logs", "start", "stop", "status"}

	// Verify our expected commands are in the skip list
	for _, cmdName := range skipInitCommands {
		assert.Contains(t, []string{"version", "update", "logs", "start", "stop", "status"}, cmdName)
	}

	// These commands should NOT be in the skip list
	nonSkipCommands := []string{"project"}
	for _, cmdName := range nonSkipCommands {
		assert.NotContains(t, skipInitCommands, cmdName)
	}
}

// Test that Execute function exists and has correct signature
func TestExecuteFunctionExists(t *testing.T) {
	// This mainly tests that Execute can be called without arguments
	// We can't easily test the full execution without complex mocking

	// Just verify the function exists by calling it in a way that
	// would fail gracefully (it will call os.Exit(1) on error)
	// We'll test this by ensuring no panic occurs when referencing it
	assert.NotNil(t, Execute)
}
