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
		Long:  "Display all Docker Compose projects currently managed by Oar with their basic information.",
		Run: func(cmd *cobra.Command, args []string) {
			projects, err := app.GetProjectService().ListProjects()
			if err != nil {
				utils.HandleCommandError("listing projects", err)
				return
			}

			if len(projects) == 0 {
				output.PrintMessage(output.Plain, "No projects found.")
				return
			}

			if err := output.PrintProjectList(projects); err != nil {
				utils.HandleCommandError("printing project list table", err)
			}
		},
	}
}
