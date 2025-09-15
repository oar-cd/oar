package project

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/oar-cd/oar/app"
	"github.com/oar-cd/oar/cmd/output"
	"github.com/spf13/cobra"
)

func NewCmdProjectDeploy() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy <project-id>",
		Short: "Deploy or update a project",
		Long: `Pull the latest changes from Git and deploy the project using Docker Compose.
This will update running containers with the latest configuration.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := runProjectDeploy(cmd, args)
			if err != nil {
				// Silence usage for runtime errors (not argument validation errors)
				cmd.SilenceUsage = true
			}
			return err
		},
	}

	cmd.Flags().Bool("pull", true, "Pull latest Git changes before deployment")
	return cmd
}

// runProjectDeploy handles the main logic for project deployment
func runProjectDeploy(cmd *cobra.Command, args []string) error {
	projectID, err := uuid.Parse(args[0])
	if err != nil {
		return fmt.Errorf("invalid project ID '%s': must be a valid UUID", args[0])
	}

	// Get flags
	pull, _ := cmd.Flags().GetBool("pull")

	// Get services
	projectService := app.GetProjectService()

	// Fetch project details for display
	project, err := projectService.Get(projectID)
	if err != nil {
		return fmt.Errorf("failed to find project %s: %w", projectID, err)
	}

	// Display deployment info
	if err := output.FprintPlain(cmd, "Starting deployment for project '%s'\n", project.Name); err != nil {
		return err
	}
	if err := output.FprintPlain(cmd, "Git URL: %s", project.GitURL); err != nil {
		return err
	}

	if pull {
		if err := output.FprintPlain(cmd, "Git pull: enabled\n"); err != nil {
			return err
		}
	} else {
		if err := output.FprintPlain(cmd, "Git pull: disabled\n"); err != nil {
			return err
		}
	}

	// Handle git pull messaging
	if pull {
		if err := output.FprintPlain(cmd, "Pulling latest changes from Git...\n"); err != nil {
			return err
		}
	}

	// Deploy project with direct stdout/stderr piping
	err = projectService.DeployPiping(projectID, pull)
	if err != nil {
		return err
	}

	// Get updated project for final status
	updatedProject, err := projectService.Get(projectID)
	if err != nil {
		return fmt.Errorf("failed to get updated project status: %w", err)
	}

	if err := output.FprintSuccess(cmd, "\nProject '%s' deployed successfully\n", updatedProject.Name); err != nil {
		return err
	}
	if err := output.FprintPlain(cmd, "Status: %s", updatedProject.Status.String()); err != nil {
		return err
	}

	if updatedProject.LastCommit != nil {
		shortCommit := *updatedProject.LastCommit
		if len(shortCommit) > 8 {
			shortCommit = shortCommit[:8]
		}
		if err := output.FprintPlain(cmd, "Latest commit: %s", shortCommit); err != nil {
			return err
		}
	}

	return nil
}
