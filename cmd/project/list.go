package project

import (
	"os"

	"github.com/ch00k/oar/cmd/output"
	"github.com/ch00k/oar/cmd/utils"
	"github.com/ch00k/oar/internal/app"
	"github.com/olekukonko/tablewriter"
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

			table := tablewriter.NewWriter(os.Stdout)
			table.Header([]string{"ID", "Name"})

			var data [][]string

			for _, project := range projects {
				data = append(data, []string{project.ID.String(), project.Name})
			}

			if err := table.Bulk(data); err != nil {
				utils.HandleCommandError("rendering project table", err)
				return
			}

			if err := table.Render(); err != nil {
				utils.HandleCommandError("rendering project table", err)
				return
			}
		},
	}
}
