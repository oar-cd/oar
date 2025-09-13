package project

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/oar-cd/oar/cmd/output"
	"github.com/oar-cd/oar/internal/app"
	"github.com/spf13/cobra"
)

func NewCmdProjectStop() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop <project-id>",
		Short: "Stop a running project",
		Long: `Stop a running Docker Compose project.
This will gracefully shut down all containers associated with the project.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := runProjectStop(cmd, args)
			if err != nil {
				// Silence usage for runtime errors (not argument validation errors)
				cmd.SilenceUsage = true
			}
			return err
		},
	}

	return cmd
}

// runProjectStop handles the main logic for project stopping
func runProjectStop(cmd *cobra.Command, args []string) error {
	projectID, err := uuid.Parse(args[0])
	if err != nil {
		return fmt.Errorf("invalid project ID '%s': must be a valid UUID", args[0])
	}

	// Get services
	projectService := app.GetProjectService()

	// Fetch project details for display
	project, err := projectService.Get(projectID)
	if err != nil {
		return fmt.Errorf("failed to find project %s: %w", projectID, err)
	}

	// Display stop info
	if err := output.FprintPlain(cmd, "Stopping project '%s'\n", project.Name); err != nil {
		return err
	}

	// Stop project with direct stdout/stderr piping
	err = projectService.StopPiping(projectID)
	if err != nil {
		return err
	}

	// Get updated project for final status
	updatedProject, err := projectService.Get(projectID)
	if err != nil {
		return fmt.Errorf("failed to get updated project status: %w", err)
	}

	if err := output.FprintSuccess(cmd, "\nProject '%s' stopped successfully\n", updatedProject.Name); err != nil {
		return err
	}
	if err := output.FprintPlain(cmd, "Status: %s", updatedProject.Status.String()); err != nil {
		return err
	}

	return nil
}
