package project

import (
	"fmt"

	"github.com/ch00k/oar/cmd/output"
	"github.com/ch00k/oar/internal/app"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

func NewCmdProjectConfig() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config <project-id>",
		Short: "Show the Docker Compose configuration for a project",
		Long: `Display the generated Docker Compose configuration for a project.
This shows the final configuration after resolving all variables and includes.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProjectConfig(cmd, args)
		},
	}

	return cmd
}

// runProjectConfig handles the main logic for displaying project configuration
func runProjectConfig(cmd *cobra.Command, args []string) error {
	projectID, err := uuid.Parse(args[0])
	if err != nil {
		return fmt.Errorf("invalid project ID '%s': must be a valid UUID", args[0])
	}

	// Get services
	projectService := app.GetProjectService()

	// Fetch project details for display
	project, err := projectService.Get(projectID)
	if err != nil {
		return fmt.Errorf("failed to find project %s: %w", projectID, err)
	}

	// Display config info
	if err := output.FprintPlain(cmd, "Docker Compose configuration for project '%s'\n", project.Name); err != nil {
		return err
	}

	// Get configuration
	config, err := projectService.GetConfig(projectID)
	if err != nil {
		return fmt.Errorf("failed to get project configuration: %w", err)
	}

	// Output the raw YAML configuration
	if err := output.FprintPlain(cmd, "%s", config); err != nil {
		return err
	}

	return nil
}
