// Package stop provides the stop command for stopping the Oar web service.
package stop

import (
	"fmt"

	"github.com/oar-cd/oar/cmd/output"
	"github.com/oar-cd/oar/cmd/utils"
	"github.com/spf13/cobra"
)

// NewCmdStop creates the stop command
func NewCmdStop() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop the Oar web service",
		Long: `Stop the Oar web service container.
This is equivalent to running 'docker compose down' in the Oar installation directory.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStop(cmd)
		},
	}

	return cmd
}

func runStop(cmd *cobra.Command) error {
	// Create ComposeProject for the Oar service
	oarComposeProject, err := utils.CreateOarServiceComposeProject(cmd)
	if err != nil {
		return err
	}

	if err := output.FprintPlain(cmd, "Stopping Oar service...\n"); err != nil {
		return err
	}

	// Use the existing DownPiping method for direct stdout/stderr piping
	if err := oarComposeProject.DownPiping(); err != nil {
		return fmt.Errorf("failed to stop Oar service: %w", err)
	}

	if err := output.FprintSuccess(cmd, "Oar service stopped successfully"); err != nil {
		return err
	}

	return nil
}
