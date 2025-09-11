package project

import (
	"fmt"
	"os"

	"github.com/ch00k/oar/cmd/output"
	"github.com/ch00k/oar/cmd/utils"
	"github.com/ch00k/oar/internal/app"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

func NewCmdProjectStop() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop <project-id>",
		Short: "Stop a running project",
		Long: `Stop a running Docker Compose project.
This will gracefully shut down all containers associated with the project.`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := runProjectStop(cmd, args); err != nil {
				utils.HandleCommandError("stopping project", err)
				os.Exit(1)
			}
		},
	}

	return cmd
}

// runProjectStop handles the main logic for project stopping
func runProjectStop(cmd *cobra.Command, args []string) error {
	projectID, err := uuid.Parse(args[0])
	if err != nil {
		utils.HandleInvalidUUID("project stop", args[0])
		return nil // This won't be reached due to os.Exit(1) in HandleInvalidUUID
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
