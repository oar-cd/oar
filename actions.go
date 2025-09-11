package main

import (
	"net/http"

	"github.com/ch00k/oar/internal/app"
	"github.com/google/uuid"
)

// Project action functions

// createProject handles project creation
func createProject(r *http.Request) error {
	// Extract form data into request struct
	req := &ProjectCreateRequest{
		Name:         r.FormValue("name"),
		GitURL:       r.FormValue("git_url"),
		GitBranch:    r.FormValue("git_branch"),
		ComposeFiles: r.FormValue("compose_files"),
		Variables:    r.FormValue("variables"),
		GitAuth:      buildGitAuthConfig(r),
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

// updateProject handles project updates
func updateProject(r *http.Request) error {
	projectID, err := parseProjectID(r)
	if err != nil {
		return err
	}

	// Extract form data into request struct
	req := &ProjectUpdateRequest{
		ID:           projectID,
		Name:         r.FormValue("name"),
		ComposeFiles: r.FormValue("compose_files"),
		Variables:    r.FormValue("variables"),
		GitAuth:      buildGitAuthConfig(r),
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

// deleteProject handles project deletion
func deleteProject(r *http.Request) error {
	projectID, err := parseProjectID(r)
	if err != nil {
		return err
	}

	projectService := app.GetProjectService()
	return projectService.Remove(projectID)
}

// Streaming action functions

// deployProject handles project deployment streaming
func deployProject(projectID uuid.UUID, outputChan chan<- string) error {
	projectService := app.GetProjectService()
	return projectService.DeployStreaming(projectID, true, outputChan)
}

// stopProject handles project stop streaming
func stopProject(projectID uuid.UUID, outputChan chan<- string) error {
	projectService := app.GetProjectService()
	return projectService.StopStreaming(projectID, outputChan)
}

// getProjectLogs handles project logs streaming
func getProjectLogs(projectID uuid.UUID, outputChan chan<- string) error {
	projectService := app.GetProjectService()
	return projectService.GetLogsStreaming(projectID, outputChan)
}
