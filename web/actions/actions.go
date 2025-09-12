// Package actions provides business action handlers for web operations.
package actions

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/oar-cd/oar/internal/app"
	"github.com/oar-cd/oar/web/handlers"
)

// Project action functions

// CreateProject handles project creation
func CreateProject(r *http.Request) error {
	// Extract form data into request struct
	req := &ProjectCreateRequest{
		Name:           r.FormValue("name"),
		GitURL:         r.FormValue("git_url"),
		GitBranch:      r.FormValue("git_branch"),
		ComposeFiles:   r.FormValue("compose_files"),
		Variables:      r.FormValue("variables"),
		GitAuth:        handlers.BuildGitAuthConfig(r),
		WatcherEnabled: r.FormValue("watcher_enabled") == "on",
	}

	// Validate request
	if err := validateProjectCreateRequest(req); err != nil {
		return err
	}

	// Build project from request
	newProject := buildProjectFromCreateRequest(req)

	// Create project using service
	projectService := app.GetProjectService()
	_, err := projectService.Create(newProject)
	return err
}

// UpdateProject handles project updates
func UpdateProject(r *http.Request) error {
	projectID, err := handlers.ParseProjectID(r)
	if err != nil {
		return err
	}

	// Extract form data into request struct
	req := &ProjectUpdateRequest{
		ID:             projectID,
		Name:           r.FormValue("name"),
		ComposeFiles:   r.FormValue("compose_files"),
		Variables:      r.FormValue("variables"),
		GitAuth:        handlers.BuildGitAuthConfig(r),
		WatcherEnabled: r.FormValue("watcher_enabled") == "on",
	}

	// Validate request
	if err := validateProjectUpdateRequest(req); err != nil {
		return err
	}

	// Get existing project
	projectService := app.GetProjectService()
	existingProject, err := projectService.Get(projectID)
	if err != nil {
		return err
	}

	// Apply updates to existing project
	applyProjectUpdateRequest(existingProject, req)

	// Update project using service
	return projectService.Update(existingProject)
}

// DeleteProject handles project deletion
func DeleteProject(r *http.Request) error {
	projectID, err := handlers.ParseProjectID(r)
	if err != nil {
		return err
	}

	projectService := app.GetProjectService()
	return projectService.Remove(projectID)
}

// Streaming action functions

// DeployProject handles project deployment streaming
func DeployProject(projectID uuid.UUID, outputChan chan<- string) error {
	projectService := app.GetProjectService()
	return projectService.DeployStreaming(projectID, true, outputChan)
}

// StopProject handles project stop streaming
func StopProject(projectID uuid.UUID, outputChan chan<- string) error {
	projectService := app.GetProjectService()
	return projectService.StopStreaming(projectID, outputChan)
}

// GetProjectLogs handles project logs streaming
func GetProjectLogs(projectID uuid.UUID, outputChan chan<- string) error {
	projectService := app.GetProjectService()
	return projectService.GetLogsStreaming(projectID, outputChan)
}
