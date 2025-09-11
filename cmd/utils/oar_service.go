package utils

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ch00k/oar/cmd/output"
	"github.com/ch00k/oar/services"
	"github.com/spf13/cobra"
)

// CreateOarServiceComposeProject creates a ComposeProject for the Oar service itself
// This is used by commands that need to manage the Oar Docker container (logs, start, stop)
func CreateOarServiceComposeProject(cmd *cobra.Command) (*services.ComposeProject, error) {
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
