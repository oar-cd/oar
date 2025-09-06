// Package start provides the start command for starting the Oar web service.
package start

import (
	"fmt"

	"github.com/ch00k/oar/cmd/output"
	"github.com/ch00k/oar/cmd/utils"
	"github.com/spf13/cobra"
)

// NewCmdStart creates the start command
func NewCmdStart() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the Oar web service",
		Long: `Start the Oar web service container.
This is equivalent to running 'docker compose up -d' in the Oar installation directory.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStart(cmd)
		},
	}

	return cmd
}

func runStart(cmd *cobra.Command) error {
	// Create ComposeProject for the Oar service
	oarComposeProject, err := utils.CreateOarServiceComposeProject(cmd)
	if err != nil {
		return err
	}

	if err := output.FprintPlain(cmd, "Starting Oar service...\n"); err != nil {
		return err
	}

	// Use the existing UpPiping method for direct stdout/stderr piping
	if err := oarComposeProject.UpPiping(); err != nil {
		return fmt.Errorf("failed to start Oar service: %w", err)
	}

	if err := output.FprintSuccess(cmd, "Oar service started successfully"); err != nil {
		return err
	}

	return nil
}
