package project

import (
	"github.com/ch00k/oar/cmd/output"
	"github.com/ch00k/oar/cmd/utils"
	"github.com/ch00k/oar/internal/app"
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
		Run: func(cmd *cobra.Command, args []string) {
			projects, err := app.GetProjectService().List()
			if err != nil {
				utils.HandleCommandError("listing projects", err)
				return
			}

			if len(projects) == 0 {
				if err := output.FprintPlain(cmd, "No projects found."); err != nil {
					utils.HandleCommandError("printing no projects message", err)
				}
				return
			}

			out, err := output.PrintProjectList(projects)
			if err != nil {
				utils.HandleCommandError("printing project list table", err)
				return
			}

			if err := output.FprintPlain(cmd, "%s", out); err != nil {
				utils.HandleCommandError("printing project list output", err)
			}
		},
	}
}
