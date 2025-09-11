package project

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/ch00k/oar/cmd/output"
	"github.com/ch00k/oar/internal/app"
	"github.com/ch00k/oar/services"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

func NewCmdProjectRemove() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove <project-id>",
		Short: "Remove a project and its data",
		Long: `Remove a project from Oar management.

This operation will permanently delete:
- Local repository clone and all Git data
- All deployment history and logs
- Project configuration and metadata

The project cannot be recovered after deletion.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProjectRemove(cmd, args)
		},
	}

	// Add confirmation flags
	cmd.Flags().BoolP("confirm", "y", false, "Skip confirmation prompt and proceed with deletion")
	cmd.Flags().Bool("force", false, "Force removal even if project is running")

	return cmd
}

// runProjectRemove handles the main logic for project removal
func runProjectRemove(cmd *cobra.Command, args []string) error {
	projectID, err := uuid.Parse(args[0])
	if err != nil {
		return fmt.Errorf("invalid project ID '%s': must be a valid UUID", args[0])
	}

	// Get confirmation flags
	skipConfirmation, _ := cmd.Flags().GetBool("confirm")
	forceRemoval, _ := cmd.Flags().GetBool("force")

	// Fetch project details before removal
	project, err := app.GetProjectService().Get(projectID)
	if err != nil {
		return fmt.Errorf("failed to find project %s: %w", projectID, err)
	}

	// Display project information for confirmation
	if err := output.FprintWarning(cmd, "\nWARNING: You are about to DELETE the following project:\n"); err != nil {
		return err
	}

	// Show project details
	projectInfo, err := output.PrintProjectDetails(project, true)
	if err != nil {
		return fmt.Errorf("failed to format project details: %w", err)
	}
	if err := output.FprintPlain(cmd, "%s\n", projectInfo); err != nil {
		return err
	}

	// Check if project is running and warn
	if project.Status == services.ProjectStatusRunning {
		if !forceRemoval {
			if err := output.FprintError(cmd, "ERROR: Project is currently RUNNING!\n"); err != nil {
				return err
			}
			if err := output.FprintPlain(cmd, "Stop the project first, or use --force to override.\n"); err != nil {
				return err
			}
			return fmt.Errorf("cannot remove running project without --force flag")
		} else {
			if err := output.FprintWarning(cmd, "WARNING: Project is RUNNING but will be force-removed\n"); err != nil {
				return err
			}
		}
	}

	// Show what will be deleted
	if err := output.FprintWarning(cmd, "\nThis will permanently delete:\n"); err != nil {
		return err
	}
	if err := output.FprintPlain(cmd, "Repository clone: %s\n", project.WorkingDir); err != nil {
		return err
	}
	if err := output.FprintPlain(cmd, "All deployment history and logs\n"); err != nil {
		return err
	}
	if err := output.FprintPlain(cmd, "Project configuration and metadata\n\n"); err != nil {
		return err
	}

	// Confirmation prompt (unless skipped)
	if !skipConfirmation {
		if !promptConfirmation(cmd, project.Name) {
			if err := output.FprintPlain(cmd, "Project removal cancelled.\n"); err != nil {
				return err
			}
			return nil
		}
	}

	// Perform the removal
	if err := output.FprintPlain(cmd, "Removing project...\n"); err != nil {
		return err
	}

	if err := app.GetProjectService().Remove(projectID); err != nil {
		return fmt.Errorf("failed to remove project: %w", err)
	}

	if err := output.FprintSuccess(cmd, "Project '%s' removed successfully\n", project.Name); err != nil {
		return err
	}

	return nil
}

// promptConfirmation asks the user to confirm project deletion
func promptConfirmation(cmd *cobra.Command, projectName string) bool {
	if err := output.FprintWarning(cmd, "Type the project name '%s' to confirm deletion: ", projectName); err != nil {
		return false
	}

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	input = strings.TrimSpace(input)
	return input == projectName
}
