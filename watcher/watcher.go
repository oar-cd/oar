// Package watcher provides the watcher service for automatic project deployments.
package watcher

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/oar-cd/oar/services"
)

type WatcherService struct {
	projectService services.ProjectManager
	gitService     services.GitExecutor
	pollInterval   time.Duration
}

func NewWatcherService(
	projectService services.ProjectManager,
	gitService services.GitExecutor,
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
		if project.WatcherEnabled {
			if project.Status == services.ProjectStatusStopped {
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

func (w *WatcherService) checkProject(ctx context.Context, project *services.Project) error {
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

	// Compare with current commit
	if currentCommit != remoteCommit {
		slog.Info("New commit detected, triggering automatic deployment",
			"project_id", project.ID,
			"project_name", project.Name,
			"old_commit", currentCommit,
			"new_commit", remoteCommit)

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
			"deployed_commit", remoteCommit)
	}

	return nil
}
