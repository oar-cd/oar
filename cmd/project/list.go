package project

import (
	"github.com/oar-cd/oar/cmd/output"
	"github.com/oar-cd/oar/internal/app"
	"github.com/spf13/cobra"
)

func NewCmdProjectList() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all managed projects",
		Long: `Display all Docker Compose projects currently managed by Oar.

Shows project information in a table format including:
- Project name and current status (with color coding)
- Git repository URL and latest commit hash
- Creation and update timestamps`,
		RunE: func(cmd *cobra.Command, args []string) error {
			projects, err := app.GetProjectService().List()
			if err != nil {
				return err
			}

			if len(projects) == 0 {
				if err := output.FprintPlain(cmd, "No projects found."); err != nil {
					return err
				}
				return nil
			}

			out, err := output.PrintProjectList(projects)
			if err != nil {
				return err
			}

			if err := output.FprintPlain(cmd, "%s", out); err != nil {
				return err
			}

			return nil
		},
	}
}
