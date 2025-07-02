package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/ch00k/oar/internal/app"
	"github.com/ch00k/oar/services"
	"github.com/google/uuid"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

// handleCommandError provides consistent error handling for CLI commands
func handleCommandError(operation string, err error, context ...any) {
	slog.Error("Command failed", append([]any{"operation", operation, "error", err}, context...)...)
	printMessage(Error, "Error: %s failed: %v", operation, err)
}

// handleInvalidUUID provides consistent handling for invalid UUID errors
func handleInvalidUUID(operation, input string) {
	slog.Warn("Invalid UUID provided", "operation", operation, "input", input)
	printMessage(Error, "Error: Invalid project ID '%s'. Must be a valid UUID.", input)
}

// projectCmd represents the project command
var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage Docker Compose projects",
}

// projectListCmd represents the command to list projects
var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all managed projects",
	Long:  "Display all Docker Compose projects currently managed by Oar with their basic information.",
	Run: func(cmd *cobra.Command, args []string) {
		projects, err := app.GetProjectService().ListProjects()
		if err != nil {
			handleCommandError("listing projects", err)
			return
		}

		if len(projects) == 0 {
			printMessage(Plain, "No projects found.")
			return
		}

		table := tablewriter.NewWriter(os.Stdout)
		table.Header([]string{"ID", "Name"})

		var data [][]string

		for _, project := range projects {
			data = append(data, []string{project.ID.String(), project.Name})
		}

		if err := table.Bulk(data); err != nil {
			handleCommandError("rendering project table", err)
			return
		}

		if err := table.Render(); err != nil {
			handleCommandError("rendering project table", err)
			return
		}
	},
}

// projectAddCmd represents the command to add a new project
var projectAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a Git repository as a managed project",
	Long: `Add a new Docker Compose project from a Git repository.
Oar will clone the repository and detect Docker Compose files automatically.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Get flag values
		gitURL, _ := cmd.Flags().GetString("git-url")
		name, _ := cmd.Flags().GetString("name")
		composeFiles, _ := cmd.Flags().GetStringArray("compose-file")
		envFiles, _ := cmd.Flags().GetStringArray("env-file")

		// Create project struct from CLI input
		project := services.NewProject(name, gitURL, composeFiles, envFiles)

		// Call service
		if err := app.GetProjectService().CreateProject(project); err != nil {
			handleCommandError("creating project from %s", err, "Git URL", gitURL)
			return
		}

		printMessage(Success, "Created project: %s (ID: %s)", project.Name, project.ID)
	},
}

// projectRemoveCmd represents the command to remove a project
var projectRemoveCmd = &cobra.Command{
	Use:   "remove <project-id>",
	Short: "Remove a project and its data",
	Long: `Remove a project from Oar management.
This deletes the local repository clone and all deployment history.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		projectID, err := uuid.Parse(args[0])
		if err != nil {
			handleInvalidUUID("project operation", args[0])
			return
		}
		printMessage(Warning, "Removing project with ID: %s", projectID)

		if err := app.GetProjectService().RemoveProject(projectID); err != nil {
			handleCommandError("removing project", err, "project_id", projectID)
			return
		}

		printMessage(Success, "Project removed successfully")
	},
}

// projectShowCmd represents the command to show project details
var projectShowCmd = &cobra.Command{
	Use:   "show <project-id>",
	Short: "Show detailed project information",
	Long:  "Display comprehensive information about a project including configuration, deployment history, and current status.",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			if err := cmd.Help(); err != nil {
				handleCommandError("showing help", err)
			}
			return
		}

		projectID, err := uuid.Parse(args[0])
		if err != nil {
			handleInvalidUUID("project operation", args[0])
			return
		}

		project, err := app.GetProjectService().GetProject(projectID)
		if err != nil {
			handleCommandError("retrieving project", err, "project_id", projectID)
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
			handleCommandError("rendering project table", err)
			return
		}

		if err := table.Render(); err != nil {
			handleCommandError("rendering project table", err)
			return
		}
	},
}

var projectDeployCmd = &cobra.Command{
	Use:   "deploy <project-id>",
	Short: "Deploy or update a project",
	Long: `Pull the latest changes from Git and deploy the project using Docker Compose.
This will update running containers with the latest configuration.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Parse UUID
		projectID, err := uuid.Parse(args[0])
		if err != nil {
			handleInvalidUUID("project operation", args[0])
			return
		}

		// Get flags
		pull, _ := cmd.Flags().GetBool("pull")

		// Deploy project
		deployment, err := app.GetProjectService().DeployProject(projectID, pull)
		if err != nil {
			handleCommandError("deploying project", err, "project_id", projectID)
			return
		}

		printMessage(Success, "Deployment started successfully!")
		printMessage(Plain, "Deployment ID: %s", deployment.ID)
		// printMessage(Plain, "Project: %s", deployment.Project.Name)
		printMessage(Plain, "Status: %s", deployment.Status)
		printMessage(Plain, "Use 'oar deployment logs %s' to view output", deployment.ID)
	},
}

func init() {
	projectCmd.AddCommand(projectListCmd)

	projectAddCmd.Flags().StringP("git-url", "u", "", "Git repository URL")
	projectAddCmd.Flags().StringP("name", "n", "", "Custom project name (auto-detected if not specified)")
	projectAddCmd.Flags().
		StringArrayP("compose-file", "f", nil, "Docker Compose file path (relative to repository root)")
	projectAddCmd.Flags().StringArrayP("env-file", "e", nil, "Environment file path (absolute)")
	if err := projectAddCmd.MarkFlagRequired("git-url"); err != nil {
		slog.Error("Failed to mark git-url flag as required", "error", err)
		panic(fmt.Sprintf("CLI setup error: %v", err)) // This is a setup error, should panic
	}

	projectCmd.AddCommand(projectAddCmd)

	projectCmd.AddCommand(projectRemoveCmd)
	projectCmd.AddCommand(projectShowCmd)

	projectDeployCmd.Flags().Bool("pull", true, "Pull latest Git changes before deployment")
	projectCmd.AddCommand(projectDeployCmd)

	rootCmd.AddCommand(projectCmd)
}
