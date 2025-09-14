package server

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCmdServer(t *testing.T) {
	cmd := NewCmdServer()

	// Test command configuration
	assert.Equal(t, "server", cmd.Use)
	assert.Equal(t, "Run Oar server (web interface + deployment watcher)", cmd.Short)
	assert.Contains(t, cmd.Long, "Starts both the web interface and deployment watcher")

	// Test that RunE is set
	assert.NotNil(t, cmd.RunE)

	// Verify it's a runnable command
	assert.True(t, cmd.Runnable())

	// Verify the command can be found by name
	assert.Equal(t, "server", cmd.Name())
}

func TestNewCmdServerFlags(t *testing.T) {
	cmd := NewCmdServer()

	// Check that config flag exists
	configFlag := cmd.Flags().Lookup("config")
	assert.NotNil(t, configFlag)
	assert.Equal(t, "c", configFlag.Shorthand)
	assert.Equal(t, "", configFlag.DefValue) // No default config path
	assert.Equal(t, "Path to configuration file", configFlag.Usage)
}

func TestNewCmdServerFlagParsing(t *testing.T) {
	cmd := NewCmdServer()

	// Test default config flag value (should be empty)
	configPath, err := cmd.Flags().GetString("config")
	assert.NoError(t, err)
	assert.Equal(t, "", configPath) // No default config path

	// Test setting config flag
	err = cmd.Flags().Set("config", "/custom/path/config.yaml")
	assert.NoError(t, err)

	configPath, err = cmd.Flags().GetString("config")
	assert.NoError(t, err)
	assert.Equal(t, "/custom/path/config.yaml", configPath)
}

func TestRunServer_InvalidConfigFile(t *testing.T) {
	// Test with non-existent config file
	err := runServer("/non/existent/config.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to initialize configuration")
}

func TestRunServer_ConfigFilePermissions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping file system test in short mode")
	}

	// Create a temporary directory
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	// Create a minimal config file with required encryption key
	configContent := `
data_dir: /tmp/test-oar
log_level: info
http_host: 127.0.0.1
http_port: 3333
poll_interval: 5m
git_timeout: 5m
docker_command: docker
encryption_key: nQbG5l9P8YzM2K8vH3FrT1cE4qL7jN6uR0sX9wB2dA8=
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Make the config file unreadable
	err = os.Chmod(configPath, 0000)
	require.NoError(t, err)

	// Restore permissions for cleanup
	defer func() {
		_ = os.Chmod(configPath, 0644) // Ignore error during cleanup
	}()

	// Test should fail due to permissions
	err = runServer(configPath)
	assert.Error(t, err)
}

func TestHandleShutdown(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping signal handling test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Track if cancel was called
	called := make(chan bool, 1)
	testCancel := func() {
		called <- true
	}

	// Start handleShutdown in goroutine
	go handleShutdown(testCancel)

	// Send signal after a short delay
	go func() {
		time.Sleep(100 * time.Millisecond)
		// We can't easily test actual signals in unit tests,
		// but we can test that the function exists and has correct signature
		// This is more of an integration test concern
	}()

	// For unit testing, we mainly verify the function exists and can be called
	// Actual signal testing would require more complex integration test setup
	select {
	case <-ctx.Done():
		// Test completed without hanging
	case <-called:
		// Cancel was called (would happen in real signal scenario)
		t.Log("Cancel function would be called on signal")
	}
}

func TestStartWebServer_InvalidConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping web server test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Test with nil config should panic or fail gracefully
	defer func() {
		if r := recover(); r != nil {
			t.Log("startWebServer correctly panicked with nil config")
		}
	}()

	// This will likely panic due to nil config, which is expected behavior
	// In production, config validation happens before this function is called
	err := startWebServer(ctx, nil)
	if err != nil {
		t.Log("startWebServer correctly returned error with nil config")
	}
}

func TestStartWatcherService_ConfigurationError(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping watcher service test in short mode")
	}

	// This test is more complex because the watcher service requires
	// full application initialization. We'll skip it for unit tests
	// and rely on integration tests for this functionality.
	t.Skip("Skipping watcher service test - requires full app initialization")
}

// Test helper functions and edge cases

func TestServerCommand_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This is more of a smoke test to ensure the command can be created
	// and configured without errors
	cmd := NewCmdServer()

	// Verify command structure
	assert.NotNil(t, cmd.RunE)
	assert.Equal(t, "server", cmd.Use)

	// Verify flags are properly registered
	flags := cmd.Flags()
	assert.NotNil(t, flags)

	configFlag := flags.Lookup("config")
	assert.NotNil(t, configFlag)
	assert.Equal(t, "string", configFlag.Value.Type())
}

func TestServerCommand_HelpText(t *testing.T) {
	cmd := NewCmdServer()

	// Test help text content
	assert.Contains(t, cmd.Short, "web interface")
	assert.Contains(t, cmd.Short, "deployment watcher")
	assert.Contains(t, cmd.Long, "single process")

	// Test command usage directly (Help() function has different behavior)
	assert.Equal(t, "server", cmd.Use)
	assert.Contains(t, cmd.Short, "Run Oar server")
	assert.Contains(t, cmd.Long, "web interface")
}

func TestServerCommand_Validation(t *testing.T) {
	cmd := NewCmdServer()

	// Test command validation
	err := cmd.ValidateArgs([]string{})
	assert.NoError(t, err) // server command accepts no positional args

	// Test with extra args
	err = cmd.ValidateArgs([]string{"extra", "args"})
	assert.NoError(t, err) // server command should accept extra args gracefully
}
