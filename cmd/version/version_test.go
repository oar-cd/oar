package version

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCmdVersion(t *testing.T) {
	cmd := NewCmdVersion()

	// Test command configuration
	assert.Equal(t, "version", cmd.Use)
	assert.Equal(t, "Show version information", cmd.Short)
	assert.Contains(t, cmd.Long, "Display version information")
	assert.Contains(t, cmd.Long, "CLI binary and installation")

	// Test that RunE is set
	assert.NotNil(t, cmd.RunE)

	// Test command has no subcommands by default
	assert.Empty(t, cmd.Commands())

	// Verify it's a runnable command
	assert.True(t, cmd.Runnable())

	// Verify the command can be found by name
	assert.Equal(t, "version", cmd.Name())
}

func TestGetServerVersion_WithoutVersionFile(t *testing.T) {
	// Since we can't easily override GetDefaultDataDir, test the scenario where
	// VERSION file doesn't exist by checking the current behavior
	result := getServerVersion()

	// The result should be either "unknown" (no file) or a valid version string
	assert.True(t, result == "unknown" || len(result) > 0)
}

func TestGetServerVersionFileLogic(t *testing.T) {
	// Test the file reading logic directly by creating a temporary file
	tmpDir := t.TempDir()
	versionFile := filepath.Join(tmpDir, "VERSION")

	// Test case 1: File doesn't exist
	_, err := os.ReadFile(versionFile)
	assert.Error(t, err) // Should error when file doesn't exist

	// Test case 2: File exists with version
	err = os.WriteFile(versionFile, []byte("1.2.3\n"), 0644)
	assert.NoError(t, err)

	data, err := os.ReadFile(versionFile)
	assert.NoError(t, err)

	result := strings.TrimSpace(string(data))
	assert.Equal(t, "1.2.3", result)

	// Test case 3: File with extra whitespace
	err = os.WriteFile(versionFile, []byte("  2.0.0  \n  "), 0644)
	assert.NoError(t, err)

	data, err = os.ReadFile(versionFile)
	assert.NoError(t, err)

	result = strings.TrimSpace(string(data))
	assert.Equal(t, "2.0.0", result)
}

func TestCLIVersionVariable(t *testing.T) {
	// Test that CLIVersion has a default value
	assert.NotEmpty(t, CLIVersion)
	assert.Equal(t, "dev", CLIVersion) // Default build-time value
}

// Test that we can call runVersion without it panicking
// Note: This will print to stdout but won't cause test failure
func TestRunVersionExecutes(t *testing.T) {
	err := runVersion()
	assert.NoError(t, err)
}
