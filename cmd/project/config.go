package project

import (
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/oar-cd/oar/app"
	"github.com/oar-cd/oar/cmd/output"
	"github.com/oar-cd/oar/services"
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
	config, stderr, err := projectService.GetConfig(projectID)
	if err != nil {
		return fmt.Errorf("failed to get project configuration: %w", err)
	}

	// Output stderr warnings first if any, with parsing
	if stderr != "" {
		lines := strings.Split(stderr, "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				parsedLine := services.ParseComposeLogLine(line)
				fmt.Fprintf(os.Stderr, "%s\n", parsedLine)
			}
		}
	}

	// Output the raw YAML configuration
	if err := output.FprintPlain(cmd, "%s", config); err != nil {
		return err
	}

	return nil
}
