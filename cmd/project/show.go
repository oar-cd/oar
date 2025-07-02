package project

import (
	"os"
	"strings"

	"github.com/ch00k/oar/cmd/utils"
	"github.com/ch00k/oar/internal/app"
	"github.com/google/uuid"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

func NewCmdProjectShow() *cobra.Command {
	return &cobra.Command{
		Use:   "show <project-id>",
		Short: "Show detailed project information",
		Long:  "Display comprehensive information about a project including configuration, deployment history, and current status.",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 1 {
				if err := cmd.Help(); err != nil {
					utils.HandleCommandError("showing help", err)
				}
				return
			}

			projectID, err := uuid.Parse(args[0])
			if err != nil {
				utils.HandleInvalidUUID("project operation", args[0])
				return
			}

			project, err := app.GetProjectService().GetProject(projectID)
			if err != nil {
				utils.HandleCommandError("retrieving project", err, "project_id", projectID)
				return
			}

			table := tablewriter.NewWriter(os.Stdout)

			var data [][]string

			data = append(data, []string{"ID", project.ID.String()})
			data = append(data, []string{"Name", project.Name})
			data = append(data, []string{"Working Directory", project.WorkingDir})
			data = append(data, []string{"Git URL", project.GitURL})
			data = append(data, []string{"Compose Files", strings.Join(project.ComposeFiles, ", ")})
			if project.LastCommit != nil {
				data = append(data, []string{"Last Commit", *project.LastCommit})
			} else {
				data = append(data, []string{"Last Commit", "N/A"})
			}
			data = append(data, []string{"Created At", project.CreatedAt.Format("2006-01-02 15:04:05")})
			data = append(data, []string{"Updated At", project.UpdatedAt.Format("2006-01-02 15:04:05")})
			// data = append(data, []string{"Deployments", fmt.Sprintf("%d", len(project.Deployments))})
			data = append(data, []string{"Status", project.Status.String()})

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
