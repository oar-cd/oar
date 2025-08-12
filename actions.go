package main

import (
	"errors"
	"net/http"
	"strings"

	"github.com/ch00k/oar/internal/app"
	"github.com/ch00k/oar/services"
	"github.com/google/uuid"
)

// Project action functions

// createProject handles project creation
func createProject(r *http.Request) error {
	// Extract form data
	name := r.FormValue("name")
	gitURL := r.FormValue("git_url")
	composeFiles := r.FormValue("compose_files")
	variables := r.FormValue("variables")

	// Validate required fields
	if name == "" || gitURL == "" || strings.TrimSpace(composeFiles) == "" {
		return errors.New("name, Git URL, and Compose Files are required")
	}

	// Build GitAuthConfig
	gitAuthConfig := buildGitAuthConfig(r)

	// Parse compose files and variables into slices
	var composeFileList []string
	if composeFiles != "" {
		composeFileList = strings.Split(strings.TrimSpace(composeFiles), "\n")
	}

	var variableList []string
	if variables != "" {
		variableList = strings.Split(strings.TrimSpace(variables), "\n")
	}

	// Create project struct
	newProject := &services.Project{
		ID:           uuid.New(),
		Name:         name,
		GitURL:       gitURL,
		GitAuth:      gitAuthConfig,
		ComposeFiles: composeFileList,
		Variables:    variableList,
		Status:       services.ProjectStatusStopped,
	}

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

	// Extract form data
	name := r.FormValue("name")
	composeFiles := r.FormValue("compose_files")
	variables := r.FormValue("variables")

	// Validate required fields
	if name == "" || strings.TrimSpace(composeFiles) == "" {
		return errors.New("name and Compose Files are required")
	}

	// Build GitAuthConfig
	gitAuthConfig := buildGitAuthConfig(r)

	// Parse compose files and variables into slices
	var composeFileList []string
	if composeFiles != "" {
		composeFileList = strings.Split(strings.TrimSpace(composeFiles), "\n")
	}

	var variableList []string
	if variables != "" {
		variableList = strings.Split(strings.TrimSpace(variables), "\n")
	}

	// Get existing project
	projectService := app.GetProjectService()
	existingProject, err := projectService.Get(projectID)
	if err != nil {
		return err
	}

	// Update project fields
	existingProject.Name = name
	existingProject.GitAuth = gitAuthConfig
	existingProject.ComposeFiles = composeFileList
	existingProject.Variables = variableList

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
