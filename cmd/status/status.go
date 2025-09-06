// Package status provides the status command for checking Oar web service status.
package status

import (
	"fmt"

	"github.com/ch00k/oar/cmd/output"
	"github.com/ch00k/oar/cmd/utils"
	"github.com/spf13/cobra"
)

// NewCmdStatus creates the status command
func NewCmdStatus() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show the status of the Oar web service",
		Long: `Show the status of the Oar web service container.
This displays whether the Oar service is running, uptime, and other status information.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(cmd)
		},
	}

	return cmd
}

func runStatus(cmd *cobra.Command) error {
	// Create ComposeProject for the Oar service
	oarComposeProject, err := utils.CreateOarServiceComposeProject(cmd)
	if err != nil {
		return err
	}

	// Get status
	projectStatus, err := oarComposeProject.Status()
	if err != nil {
		return fmt.Errorf("failed to get Oar service status: %w", err)
	}

	// Display status in plain text
	if err := output.FprintPlain(cmd, fmt.Sprintf("Status: %s", projectStatus.Status)); err != nil {
		return err
	}

	if projectStatus.Status == "running" && projectStatus.Uptime != "" {
		if err := output.FprintPlain(cmd, fmt.Sprintf("Uptime: %s", projectStatus.Uptime)); err != nil {
			return err
		}
	}

	// Show individual container statuses
	if len(projectStatus.Containers) > 0 {
		if err := output.FprintPlain(cmd, "\nContainers:"); err != nil {
			return err
		}
		for _, container := range projectStatus.Containers {
			prefix := "[OK]"
			if container.State != "running" {
				prefix = "[ERROR]"
			}
			if err := output.FprintPlain(cmd, fmt.Sprintf("  %s %s: %s", prefix, container.Service, container.Status)); err != nil {
				return err
			}
		}
	}

	return nil
}
