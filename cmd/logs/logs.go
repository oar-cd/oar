// Package logs provides the logs command for viewing Oar web service logs.
package logs

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ch00k/oar/cmd/output"
	"github.com/ch00k/oar/services"
	"github.com/spf13/cobra"
)

// NewCmdLogs creates the logs command
func NewCmdLogs() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs",
		Short: "View logs from the Oar web service",
		Long: `Display logs from the running Oar web service container.
This follows the logs in real-time (equivalent to docker compose logs --follow).
Press Ctrl+C to stop.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogs(cmd)
		},
	}

	return cmd
}

func runLogs(cmd *cobra.Command) error {
	// Get the Oar data directory (where compose.yaml should be)
	oarDir := services.GetDefaultDataDir()

	// Check if compose.yaml exists
	composeFile := filepath.Join(oarDir, "compose.yaml")
	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		return output.FprintError(
			cmd,
			"Oar compose.yaml not found at %s\nMake sure Oar is installed and running.",
			composeFile,
		)
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

	if err := output.FprintPlain(cmd, "Streaming logs from Oar service..."); err != nil {
		return err
	}
	if err := output.FprintPlain(cmd, "Press Ctrl+C to stop\n"); err != nil {
		return err
	}

	// Use the existing LogsPiping method for direct stdout/stderr piping
	if err := oarComposeProject.LogsPiping(); err != nil {
		return fmt.Errorf("failed to get logs: %w", err)
	}

	return nil
}
