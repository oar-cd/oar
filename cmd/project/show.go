package project

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/oar-cd/oar/cmd/output"
	"github.com/oar-cd/oar/internal/app"
	"github.com/spf13/cobra"
)

func NewCmdProjectShow() *cobra.Command {
	return &cobra.Command{
		Use:   "show <project-id>",
		Short: "Show detailed project information",
		Long:  "Display comprehensive information about a project including configuration, deployment history, and current status.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return cmd.Help()
			}

			projectID, err := uuid.Parse(args[0])
			if err != nil {
				return fmt.Errorf("invalid project ID '%s': must be a valid UUID", args[0])
			}

			project, err := app.GetProjectService().Get(projectID)
			if err != nil {
				return fmt.Errorf("failed to retrieve project %s: %w", projectID, err)
			}

			out, err := output.PrintProjectDetails(project, false)
			if err != nil {
				return fmt.Errorf("failed to format project details: %w", err)
			}

			if err := output.FprintPlain(cmd, "%s", out); err != nil {
				return fmt.Errorf("failed to print project details: %w", err)
			}

			return nil
		},
	}
}
