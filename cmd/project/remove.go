package project

import (
	"github.com/ch00k/oar/cmd/output"
	"github.com/ch00k/oar/cmd/utils"
	"github.com/ch00k/oar/internal/app"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

func NewCmdProjectRemove() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <project-id>",
		Short: "Remove a project and its data",
		Long: `Remove a project from Oar management.
This deletes the local repository clone and all deployment history.`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			projectID, err := uuid.Parse(args[0])
			if err != nil {
				utils.HandleInvalidUUID("project operation", args[0])
				return
			}
			output.PrintMessage(output.Warning, "Removing project with ID: %s", projectID)

			if err := app.GetProjectService().Remove(projectID); err != nil {
				utils.HandleCommandError("removing project", err, "project_id", projectID)
				return
			}

			output.PrintMessage(output.Success, "Project removed successfully")
		},
	}
}
