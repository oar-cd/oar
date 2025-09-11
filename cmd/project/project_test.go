package project

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCmdProject(t *testing.T) {
	cmd := NewCmdProject()

	// Test command configuration
	assert.Equal(t, "project", cmd.Use)
	assert.Equal(t, "Manage Docker Compose projects", cmd.Short)

	// This is a parent command, so it shouldn't have its own RunE
	assert.Nil(t, cmd.RunE)

	// Verify the command can be found by name
	assert.Equal(t, "project", cmd.Name())

	// Test that subcommands are properly registered
	subcommands := cmd.Commands()
	assert.NotEmpty(t, subcommands)

	// Check for expected subcommands
	subcommandNames := make([]string, len(subcommands))
	for i, subcmd := range subcommands {
		subcommandNames[i] = subcmd.Name()
	}

	expectedSubcommands := []string{
		"list", "add", "remove", "show", "deploy", "stop", "status", "config", "logs",
	}

	for _, expected := range expectedSubcommands {
		assert.Contains(t, subcommandNames, expected, "Expected subcommand %s not found", expected)
	}

	// Verify we have the correct number of subcommands
	assert.Len(t, subcommands, len(expectedSubcommands))
}

func TestNewCmdProjectSubcommandTypes(t *testing.T) {
	cmd := NewCmdProject()
	subcommands := cmd.Commands()

	// Verify all subcommands have proper configuration
	for _, subcmd := range subcommands {
		assert.NotEmpty(t, subcmd.Name(), "Subcommand should have a name")
		assert.NotEmpty(t, subcmd.Use, "Subcommand %s should have Use set", subcmd.Name())
		// Note: Some subcommands may have RunE set, others may not depending on implementation
	}
}

func TestNewCmdProjectStructure(t *testing.T) {
	cmd := NewCmdProject()

	// Verify parent command is not directly runnable (no RunE)
	assert.False(t, cmd.Runnable(), "Parent project command should not be directly runnable")

	// But should have subcommands
	assert.True(t, cmd.HasSubCommands(), "Project command should have subcommands")

	// Check that help will be shown when no subcommand is provided
	assert.True(t, cmd.HasHelpSubCommands() || !cmd.Runnable())
}
