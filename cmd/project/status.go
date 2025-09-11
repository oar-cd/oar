package project

import (
	"fmt"

	"github.com/ch00k/oar/cmd/output"
	"github.com/ch00k/oar/internal/app"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

func NewCmdProjectStatus() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status <project-id>",
		Short: "Show the status of a project's containers",
		Long: `Display the current status of all containers in a project.
This shows whether the project is running, uptime, and individual container states.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProjectStatus(cmd, args)
		},
	}

	return cmd
}

// runProjectStatus handles the main logic for displaying project status
func runProjectStatus(cmd *cobra.Command, args []string) error {
	projectID, err := uuid.Parse(args[0])
	if err != nil {
		return fmt.Errorf("invalid project ID '%s': must be a valid UUID", args[0])
	}

	// Get services
	projectService := app.GetProjectService()
	if projectService == nil {
		return fmt.Errorf("application not initialized - project service is nil")
	}

	// Get status
	projectStatus, err := projectService.GetStatus(projectID)
	if err != nil {
		return fmt.Errorf("failed to get project status: %w", err)
	}

	// Display status in same format as 'oar status'
	if err := output.FprintPlain(cmd, "Status: %s", projectStatus.Status); err != nil {
		return err
	}

	if projectStatus.Status == "running" && projectStatus.Uptime != "" {
		if err := output.FprintPlain(cmd, "Uptime: %s", projectStatus.Uptime); err != nil {
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
			if err := output.FprintPlain(cmd, "  %s %s: %s", prefix, container.Service, container.Status); err != nil {
				return err
			}
		}
	}

	return nil
}
