package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/ch00k/oar/internal/app"
	"github.com/ch00k/oar/services"
	"github.com/google/uuid"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

// handleCommandError provides consistent error handling for CLI commands
func handleCommandError(operation string, err error, context ...any) {
	slog.Error("Command failed", append([]any{"operation", operation, "error", err}, context...)...)
	printError("Error: %s failed: %v", operation, err)
}

// handleInvalidUUID provides consistent handling for invalid UUID errors
func handleInvalidUUID(operation, input string) {
	slog.Warn("Invalid UUID provided", "operation", operation, "input", input)
	printError("Error: Invalid project ID '%s'. Must be a valid UUID.", input)
}

// projectCmd represents the project command
var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage projects",
}

// projectListCmd represents the command to list projects
var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List projects",
	Long:  "Display a list of all projects with their details and current status.",
	Run: func(cmd *cobra.Command, args []string) {
		projects, err := app.GetProjectService().ListProjects()
		if err != nil {
			handleCommandError("listing projects", err)
			return
		}

		if len(projects) == 0 {
			fmt.Println("No projects found.")
			fmt.Println("Use 'oar project add <git-url>' to create your first project.")
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
	Short: "Add a new project",
	Long:  `Add a new Docker Compose project from a Git repository to be managed by Oar.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Get flag values
		gitURL, _ := cmd.Flags().GetString("git-url")
		name, _ := cmd.Flags().GetString("name")

		// Create config struct from CLI input
		config := services.CreateProjectConfig{
			GitURL: gitURL,
			Name:   name, // Could be empty string
		}

		// Call service
		project, err := app.GetProjectService().CreateProject(config)
		if err != nil {
			handleCommandError("creating project", err, "git_url", gitURL)
			return
		}

		printSuccess("Created project: %s (ID: %s)", project.Name, project.ID)
	},
}

// projectRemoveCmd represents the command to remove a project
var projectRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a project",
	Long:  `Remove a Docker Compose project from the discovery.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		projectID, err := uuid.Parse(args[0])
		if err != nil {
			handleInvalidUUID("project operation", args[0])
			return
		}
		fmt.Printf("Removing project with ID: %s\n", projectID)

		if err := app.GetProjectService().RemoveProject(projectID); err != nil {
			handleCommandError("removing project", err, "project_id", projectID)
			return
		}

		printSuccess("Project removed successfully")
	},
}

// projectShowCmd represents the command to show project details
var projectShowCmd = &cobra.Command{
	Use:   "show [id]",
	Short: "Show details of a Docker Compose project",
	Long:  `Show detailed information about a specific Docker Compose project.`,
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
		data = append(data, []string{"Git URL", project.GitURL})
		data = append(data, []string{"Compose Name", project.ComposeName})
		data = append(data, []string{"Compose File", project.ComposeFileName})
		if project.LastCommit != nil {
			data = append(data, []string{"Last Commit", *project.LastCommit})
		} else {
			data = append(data, []string{"Last Commit", "N/A"})
		}
		data = append(data, []string{"Created At", project.CreatedAt.String()})
		data = append(data, []string{"Updated At", project.UpdatedAt.String()})
		data = append(data, []string{"Deployments", fmt.Sprintf("%d", len(project.Deployments))})
		data = append(data, []string{"Status", "N/A"}) // TODO: Status not directly available in Project model

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
	Use:   "deploy project_id",
	Short: "Deploy a project",
	Long:  "Pull latest changes from Git and (re-)deploy the project using Docker Compose.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Parse UUID
		projectID, err := uuid.Parse(args[0])
		if err != nil {
			handleInvalidUUID("project operation", args[0])
			return
		}

		// Get flags
		detach, _ := cmd.Flags().GetBool("detach")
		build, _ := cmd.Flags().GetBool("build")
		pull, _ := cmd.Flags().GetBool("pull")

		// Deploy project
		deployment, err := app.GetProjectService().DeployProject(projectID, services.DeploymentConfig{
			Detach: detach,
			Build:  build,
			Pull:   pull,
		})
		if err != nil {
			handleCommandError("deploying project", err, "project_id", projectID)
			return
		}

		printSuccess("Deployment started successfully!")
		fmt.Printf("Deployment ID: %s\n", deployment.ID)
		fmt.Printf("Project: %s\n", deployment.Project.Name)
		fmt.Printf("Status: %s\n", deployment.Status)

		if !detach {
			fmt.Printf("\nDeployment Output:\n")
			fmt.Println(deployment.Output)
		} else {
			fmt.Printf("\nUse 'oar deployment logs %s' to view output\n", deployment.ID)
		}
	},
}

func init() {
	projectCmd.AddCommand(projectListCmd)

	projectAddCmd.Flags().String("git-url", "", "Git repository URL (required)")
	projectAddCmd.Flags().String("name", "", "Project name (default: derived from repo)")
	if err := projectAddCmd.MarkFlagRequired("git-url"); err != nil {
		slog.Error("Failed to mark git-url flag as required", "error", err)
		panic(fmt.Sprintf("CLI setup error: %v", err)) // This is a setup error, should panic
	}

	projectCmd.AddCommand(projectAddCmd)

	projectCmd.AddCommand(projectRemoveCmd)
	projectCmd.AddCommand(projectShowCmd)

	projectDeployCmd.Flags().BoolP("detach", "d", true, "Run in detached mode (default: true)")
	projectDeployCmd.Flags().Bool("build", false, "Build images before starting containers")
	projectDeployCmd.Flags().Bool("pull", true, "Pull latest Git changes before deploying (default: true)")
	projectCmd.AddCommand(projectDeployCmd)

	rootCmd.AddCommand(projectCmd)
}
