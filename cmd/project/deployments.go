package project

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/oar-cd/oar/app"
	"github.com/oar-cd/oar/cmd/output"
	"github.com/spf13/cobra"
)

func NewCmdProjectDeployments() *cobra.Command {
	return &cobra.Command{
		Use:   "deployments <project-id>",
		Short: "List deployments for a project",
		Long: `Display all deployments for a specific project.

Shows deployment history in a table format including:
- Deployment ID and status (with color coding)
- Commit hash and deployment timestamp
- Duration and outcome of each deployment`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return cmd.Help()
			}

			projectID, err := uuid.Parse(args[0])
			if err != nil {
				return fmt.Errorf("invalid project ID '%s': must be a valid UUID", args[0])
			}

			// First verify the project exists
			project, err := app.GetProjectService().Get(projectID)
			if err != nil {
				return fmt.Errorf("failed to retrieve project %s: %w", projectID, err)
			}

			deployments, err := app.GetProjectService().ListDeployments(projectID)
			if err != nil {
				return fmt.Errorf("failed to retrieve deployments for project %s: %w", projectID, err)
			}

			if len(deployments) == 0 {
				if err := output.FprintPlain(cmd, "No deployments found for project '%s'.", project.Name); err != nil {
					return err
				}
				return nil
			}

			out, err := output.PrintDeploymentList(deployments, project.Name)
			if err != nil {
				return fmt.Errorf("failed to format deployments: %w", err)
			}

			if err := output.FprintPlain(cmd, "%s", out); err != nil {
				return fmt.Errorf("failed to print deployments: %w", err)
			}

			return nil
		},
	}
}
