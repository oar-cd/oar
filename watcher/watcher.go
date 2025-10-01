// Package watcher provides the watcher service for automatic project deployments.
package watcher

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/oar-cd/oar/docker"
	"github.com/oar-cd/oar/domain"
	"github.com/oar-cd/oar/git"
	"github.com/oar-cd/oar/project"
)

type WatcherService struct {
	projectService project.ProjectManager
	gitService     *git.GitService
	pollInterval   time.Duration
}

func NewWatcherService(
	projectService project.ProjectManager,
	gitService *git.GitService,
	pollInterval time.Duration,
) *WatcherService {
	return &WatcherService{
		projectService: projectService,
		gitService:     gitService,
		pollInterval:   pollInterval,
	}
}

func (w *WatcherService) Start(ctx context.Context) error {
	slog.Info("Watcher service starting", "poll_interval", w.pollInterval)

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	// Run initial check immediately
	if err := w.checkAllProjects(ctx); err != nil {
		slog.Error("Initial project check failed", "error", err)
	}

	for {
		select {
		case <-ctx.Done():
			slog.Info("Watcher service shutting down")
			return nil
		case <-ticker.C:
			if err := w.checkAllProjects(ctx); err != nil {
				slog.Error("Project check failed", "error", err)
			}
		}
	}
}

func (w *WatcherService) checkAllProjects(ctx context.Context) error {
	slog.Debug("Starting project check cycle")

	projects, err := w.projectService.List()
	if err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	projectsChecked := 0
	for _, project := range projects {
		// Sync Docker status for all projects - detects mismatches and updates database
		if err := w.syncProjectStatus(ctx, project); err != nil {
			slog.Error("Failed to sync project status",
				"project_id", project.ID,
				"project_name", project.Name,
				"error", err)
		}

		// Only check git changes for watcher-enabled projects
		if project.WatcherEnabled {
			if project.Status == domain.ProjectStatusStopped {
				slog.Info("Project is stopped - skipping git check",
					"project_id", project.ID,
					"project_name", project.Name,
					"status", project.Status.String())
				continue
			}

			slog.Info("Checking project",
				"project_id", project.ID,
				"project_name", project.Name,
				"status", project.Status.String())

			projectsChecked++
			if err := w.checkProject(ctx, project); err != nil {
				slog.Error("Failed to check project",
					"project_id", project.ID,
					"project_name", project.Name,
					"error", err)
			}
		}
	}

	slog.Debug("Project check cycle completed",
		"total_projects", len(projects),
		"projects_checked", projectsChecked)

	return nil
}

func (w *WatcherService) checkProject(ctx context.Context, project *domain.Project) error {
	currentCommit := project.LastCommitStr()

	gitDir, err := project.GitDir()
	if err != nil {
		return fmt.Errorf("failed to get git directory: %w", err)
	}

	// Fetch latest changes from remote
	if err := w.gitService.Fetch(project.GitBranch, project.GitAuth, gitDir); err != nil {
		return fmt.Errorf("failed to fetch from remote: %w", err)
	}

	// Get latest commit hash from remote
	remoteCommit, err := w.gitService.GetRemoteLatestCommit(gitDir, project.GitBranch)
	if err != nil {
		return fmt.Errorf("failed to get remote commit: %w", err)
	}

	// Log git check results
	slog.Info("Git check completed",
		"project_id", project.ID,
		"project_name", project.Name,
		"current_commit", currentCommit,
		"remote_commit", remoteCommit,
		"has_updates", currentCommit != remoteCommit)

	// Deploy if there are git changes OR if project is not in running/stopped state
	hasGitChanges := currentCommit != remoteCommit
	isInErrorState := project.Status != domain.ProjectStatusRunning && project.Status != domain.ProjectStatusStopped
	shouldDeploy := hasGitChanges || isInErrorState

	if shouldDeploy {
		var reason string
		if hasGitChanges {
			reason = "new commit detected"
			slog.Info("New commit detected, triggering automatic deployment",
				"project_id", project.ID,
				"project_name", project.Name,
				"old_commit", currentCommit,
				"new_commit", remoteCommit)
		} else {
			reason = "project in error state"
			slog.Warn("Project in error state, triggering automatic deployment",
				"project_id", project.ID,
				"project_name", project.Name,
				"status", project.Status.String(),
				"target_commit", remoteCommit)
		}

		// TODO: Consider creating a dedicated method for automatic deployments
		// instead of using DeployPiping(). This would allow for:
		// - Better logging/tracking of automatic vs manual deployments
		// - Different error handling strategies
		// - Deployment throttling/rate limiting
		// - Automatic deployment-specific configuration
		if err := w.projectService.DeployPiping(project.ID, true); err != nil {
			slog.Error("Automatic deployment failed",
				"project_id", project.ID,
				"project_name", project.Name,
				"reason", reason,
				"target_commit", remoteCommit,
				"error", err)
			return fmt.Errorf("failed to deploy project: %w", err)
		}

		// Update the project's LastCommit to the newly deployed commit
		project.LastCommit = &remoteCommit
		if err := w.projectService.Update(project); err != nil {
			slog.Error("Failed to update project LastCommit after successful deployment",
				"project_id", project.ID,
				"project_name", project.Name,
				"deployed_commit", remoteCommit,
				"error", err)
			// Don't return error here as deployment was successful
			// This is just a tracking issue that won't break functionality
		} else {
			slog.Debug("Updated project LastCommit after successful deployment",
				"project_id", project.ID,
				"project_name", project.Name,
				"deployed_commit", remoteCommit)
		}

		slog.Info("Automatic deployment completed successfully",
			"project_id", project.ID,
			"project_name", project.Name,
			"reason", reason,
			"deployed_commit", remoteCommit)
	}

	return nil
}

// syncProjectStatus checks if the project's database status matches its actual Docker status and updates it if needed
func (w *WatcherService) syncProjectStatus(ctx context.Context, project *domain.Project) error {
	// Get the actual Docker status
	composeStatus, err := w.projectService.GetStatus(project.ID)
	if err != nil {
		slog.Error("Failed to get Docker status for project",
			"project_id", project.ID,
			"project_name", project.Name,
			"error", err)

		// Update database status to unknown since we can't determine actual status
		if project.Status != domain.ProjectStatusUnknown {
			slog.Warn("Updating project status to unknown due to Docker status error",
				"project_id", project.ID,
				"project_name", project.Name,
				"previous_status", project.Status.String())

			project.Status = domain.ProjectStatusUnknown
			if updateErr := w.projectService.Update(project); updateErr != nil {
				return fmt.Errorf("failed to update project status to unknown: %w", updateErr)
			}
		}
		return fmt.Errorf("failed to get Docker status: %w", err)
	}

	// Determine what the database status should be based on Docker status
	var expectedStatus domain.ProjectStatus
	switch composeStatus.Status {
	case docker.ComposeProjectStatusRunning:
		expectedStatus = domain.ProjectStatusRunning
	case docker.ComposeProjectStatusStopped:
		expectedStatus = domain.ProjectStatusStopped
	case docker.ComposeProjectStatusFailed:
		// For failed status, we'll consider it as error since containers are in mixed states
		expectedStatus = domain.ProjectStatusError
	case docker.ComposeProjectStatusUnknown:
		// For unknown status, set database to unknown as well
		expectedStatus = domain.ProjectStatusUnknown
	default:
		// Should not happen, but default to unknown
		expectedStatus = domain.ProjectStatusUnknown
	}

	// Check if there's a mismatch
	if project.Status != expectedStatus {
		slog.Warn("Project status mismatch detected - updating database",
			"project_id", project.ID,
			"project_name", project.Name,
			"database_status", project.Status.String(),
			"docker_status", composeStatus.Status.String(),
			"updating_to", expectedStatus.String())

		// Update the project status in the database
		project.Status = expectedStatus
		if err := w.projectService.Update(project); err != nil {
			return fmt.Errorf("failed to update project status: %w", err)
		}

		slog.Info("Project status updated successfully",
			"project_id", project.ID,
			"project_name", project.Name,
			"new_status", expectedStatus.String())
	}

	return nil
}
