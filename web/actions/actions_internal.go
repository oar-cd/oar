package actions

import (
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/oar-cd/oar/services"
)

// ProjectCreateRequest represents the data needed to create a project
type ProjectCreateRequest struct {
	Name           string
	GitURL         string
	GitBranch      string
	ComposeFiles   string
	Variables      string
	GitAuth        *services.GitAuthConfig
	WatcherEnabled bool
}

// ProjectUpdateRequest represents the data needed to update a project
type ProjectUpdateRequest struct {
	ID             uuid.UUID
	Name           string
	ComposeFiles   string
	Variables      string
	GitAuth        *services.GitAuthConfig
	WatcherEnabled bool
}

// validateProjectCreateRequest validates a project creation request
func validateProjectCreateRequest(req *ProjectCreateRequest) error {
	if strings.TrimSpace(req.Name) == "" {
		return errors.New("name is required")
	}
	if strings.TrimSpace(req.GitURL) == "" {
		return errors.New("git URL is required")
	}
	if strings.TrimSpace(req.ComposeFiles) == "" {
		return errors.New("compose files are required")
	}
	return nil
}

// validateProjectUpdateRequest validates a project update request
func validateProjectUpdateRequest(req *ProjectUpdateRequest) error {
	if strings.TrimSpace(req.Name) == "" {
		return errors.New("name is required")
	}
	if strings.TrimSpace(req.ComposeFiles) == "" {
		return errors.New("compose files are required")
	}
	return nil
}

// parseComposeFiles converts compose files string to slice
func parseComposeFiles(composeFiles string) []string {
	if composeFiles == "" {
		return nil
	}
	return strings.Split(strings.TrimSpace(composeFiles), "\n")
}

// parseVariables converts variables string to slice
func parseVariables(variables string) []string {
	if variables == "" {
		return nil
	}
	return strings.Split(strings.TrimSpace(variables), "\n")
}

// buildProjectFromCreateRequest converts create request to Project struct
func buildProjectFromCreateRequest(req *ProjectCreateRequest) *services.Project {
	return &services.Project{
		ID:             uuid.New(),
		Name:           req.Name,
		GitURL:         req.GitURL,
		GitBranch:      req.GitBranch,
		GitAuth:        req.GitAuth,
		ComposeFiles:   parseComposeFiles(req.ComposeFiles),
		Variables:      parseVariables(req.Variables),
		Status:         services.ProjectStatusStopped,
		WatcherEnabled: req.WatcherEnabled,
	}
}

// applyProjectUpdateRequest applies update request to existing project
func applyProjectUpdateRequest(project *services.Project, req *ProjectUpdateRequest) {
	project.Name = req.Name
	project.GitAuth = req.GitAuth
	project.ComposeFiles = parseComposeFiles(req.ComposeFiles)
	project.Variables = parseVariables(req.Variables)
	project.WatcherEnabled = req.WatcherEnabled
}
