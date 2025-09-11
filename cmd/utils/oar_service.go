// Package utils provides utility functions for CLI commands in Oar.
package utils

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/oar-cd/oar/cmd/output"
	"github.com/oar-cd/oar/services"
	"github.com/spf13/cobra"
)

// CreateOarServiceComposeProjectFunc is a function type for creating ComposeProject instances
type CreateOarServiceComposeProjectFunc func(cmd *cobra.Command) (services.ComposeProjectInterface, error)

// createOarServiceComposeProjectImpl is the default implementation
var createOarServiceComposeProjectImpl CreateOarServiceComposeProjectFunc = createOarServiceComposeProjectDefault

// CreateOarServiceComposeProject creates a ComposeProject for the Oar service itself
// This is used by commands that need to manage the Oar Docker container (logs, start, stop)
func CreateOarServiceComposeProject(cmd *cobra.Command) (services.ComposeProjectInterface, error) {
	return createOarServiceComposeProjectImpl(cmd)
}

// SetCreateOarServiceComposeProjectForTesting allows tests to inject a mock implementation
func SetCreateOarServiceComposeProjectForTesting(fn CreateOarServiceComposeProjectFunc) {
	createOarServiceComposeProjectImpl = fn
}

// ResetCreateOarServiceComposeProjectForTesting resets to the default implementation
func ResetCreateOarServiceComposeProjectForTesting() {
	createOarServiceComposeProjectImpl = createOarServiceComposeProjectDefault
}

// createOarServiceComposeProjectDefault is the default implementation
func createOarServiceComposeProjectDefault(cmd *cobra.Command) (services.ComposeProjectInterface, error) {
	// Get the Oar data directory (where compose.yaml should be)
	oarDir := services.GetDefaultDataDir()

	// Check if compose.yaml exists
	composeFile := filepath.Join(oarDir, "compose.yaml")
	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		_ = output.FprintError(
			cmd,
			"Oar compose.yaml not found at %s\nMake sure Oar is installed.",
			composeFile,
		)
		return nil, fmt.Errorf("oar compose.yaml not found at %s", composeFile)
	}

	// Create a minimal config for the docker compose operations
	config := &services.Config{
		DockerCommand: "docker",
		DockerHost:    "unix:///var/run/docker.sock",
	}

	// Create a ComposeProject for the Oar service itself
	oarComposeProject := &services.ComposeProject{
		Name:         "oar",
		WorkingDir:   oarDir,
		ComposeFiles: []string{"compose.yaml"},
		Variables:    []string{}, // No additional variables needed
		Config:       config,
	}

	return oarComposeProject, nil
}
