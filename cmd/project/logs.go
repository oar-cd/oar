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

func NewCmdProjectLogs() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs <project-id>",
		Short: "View logs from a project's containers",
		Long: `Stream logs from all containers in a Docker Compose project.
This shows real-time logs from all services in the project.`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := runProjectLogs(cmd, args); err != nil {
				utils.HandleCommandError("getting project logs", err)
				os.Exit(1)
			}
		},
	}

	return cmd
}

// runProjectLogs handles the main logic for streaming project logs
func runProjectLogs(cmd *cobra.Command, args []string) error {
	projectID, err := uuid.Parse(args[0])
	if err != nil {
		return fmt.Errorf("invalid project ID format: %s", args[0])
	}

	// Get services
	projectService := app.GetProjectService()

	// Fetch project details for display
	project, err := projectService.Get(projectID)
	if err != nil {
		return fmt.Errorf("failed to find project %s: %w", projectID, err)
	}

	// Display logs info
	if err := output.FprintPlain(cmd, "Streaming logs for project '%s'\n", project.Name); err != nil {
		return err
	}
	if err := output.FprintPlain(cmd, "Press Ctrl+C to stop\n"); err != nil {
		return err
	}

	// Stream project logs with direct stdout/stderr piping
	err = projectService.GetLogsPiping(projectID)
	if err != nil {
		return err
	}

	return nil
}
