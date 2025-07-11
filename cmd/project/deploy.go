package project

import (
	"github.com/spf13/cobra"
)

func NewCmdProjectDeploy() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy <project-id>",
		Short: "Deploy or update a project",
		Long: `Pull the latest changes from Git and deploy the project using Docker Compose.
This will update running containers with the latest configuration.`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			//// Parse UUID
			//projectID, err := uuid.Parse(args[0])
			//if err != nil {
			//    utils.HandleInvalidUUID("project operation", args[0])
			//    return
			//}

			//// Get flags
			//pull, _ := cmd.Flags().GetBool("pull")

			//// Deploy project
			//deployment, err := app.GetProjectService().DeployProject(projectID, pull)
			//if err != nil {
			//    utils.HandleCommandError("deploying project", err, "project_id", projectID)
			//    return
			//}

			//output.PrintMessage(output.Success, "Deployment started successfully!")
			//output.PrintMessage(output.Plain, "Deployment ID: %s", deployment.ID)
			//// printMessage(Plain, "Project: %s", deployment.Project.Name)
			//output.PrintMessage(output.Plain, "Status: %s", deployment.Status)
			//output.PrintMessage(output.Plain, "Use 'oar deployment logs %s' to view output", deployment.ID)
		},
	}

	cmd.Flags().Bool("pull", true, "Pull latest Git changes before deployment")
	return cmd
}
