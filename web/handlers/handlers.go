// Package handlers provides HTTP request handlers and utilities for the web server.
package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/oar-cd/oar/app"
	"github.com/oar-cd/oar/docker"
	"github.com/oar-cd/oar/domain"
	projectcomponent "github.com/oar-cd/oar/web/components/project"
)

// GetVersion returns the server version for use in templates
func GetVersion() string {
	return app.Version
}

// Helper functions for common operations

// ParseProjectID extracts and validates project ID from URL parameters
func ParseProjectID(r *http.Request) (uuid.UUID, error) {
	projectID := chi.URLParam(r, "id")
	if projectID == "" {
		return uuid.Nil, errors.New("project ID is required")
	}

	parsedID, err := uuid.Parse(projectID)
	if err != nil {
		return uuid.Nil, errors.New("invalid project ID format")
	}

	return parsedID, nil
}

// BuildGitAuthConfig creates GitAuthConfig from form values
func BuildGitAuthConfig(r *http.Request) *domain.GitAuthConfig {
	authMethod := r.FormValue("auth_method")

	switch authMethod {
	case "http":
		username := r.FormValue("username")
		password := r.FormValue("password")
		if username != "" || password != "" {
			return &domain.GitAuthConfig{
				HTTPAuth: &domain.GitHTTPAuthConfig{
					Username: username,
					Password: password,
				},
			}
		}
	case "ssh":
		sshUsername := r.FormValue("ssh_username")
		privateKey := r.FormValue("private_key")
		if privateKey != "" {
			return &domain.GitAuthConfig{
				SSHAuth: &domain.GitSSHAuthConfig{
					PrivateKey: privateKey,
					User:       sshUsername,
				},
			}
		}
	}

	return nil
}

// ConvertProjectToView converts a backend Project to frontend ProjectView
func ConvertProjectToView(p *domain.Project) projectcomponent.ProjectView {
	return projectcomponent.ProjectView{
		ID:              p.ID,
		Name:            p.Name,
		GitURL:          p.GitURL,
		GitBranch:       p.GitBranch,
		GitAuth:         ConvertGitAuthConfig(p.GitAuth),
		Status:          p.Status.String(),
		LocalCommit:     p.LocalCommit,
		ComposeFiles:    p.ComposeFiles,
		ComposeOverride: p.ComposeOverride,
		Variables:       p.Variables,
		WatcherEnabled:  p.WatcherEnabled,
		CreatedAt:       p.CreatedAt,
		UpdatedAt:       p.UpdatedAt,
	}
}

// ConvertProjectsToViews converts backend projects to frontend ProjectView
func ConvertProjectsToViews(projects []*domain.Project) []projectcomponent.ProjectView {
	views := make([]projectcomponent.ProjectView, len(projects))
	for i, p := range projects {
		views[i] = ConvertProjectToView(p)
	}
	return views
}

// ConvertGitAuthConfig converts backend GitAuthConfig to frontend GitAuthConfig
func ConvertGitAuthConfig(auth *domain.GitAuthConfig) *projectcomponent.GitAuthConfig {
	if auth == nil {
		return nil
	}

	result := &projectcomponent.GitAuthConfig{}

	if auth.HTTPAuth != nil {
		result.HTTPAuth = &projectcomponent.GitHTTPAuthConfig{
			Username: auth.HTTPAuth.Username,
			Password: auth.HTTPAuth.Password,
		}
	}

	if auth.SSHAuth != nil {
		result.SSHAuth = &projectcomponent.GitSSHAuthConfig{
			PrivateKey: auth.SSHAuth.PrivateKey,
			User:       auth.SSHAuth.User,
		}
	}

	return result
}

// renderProjectGrid fetches projects and renders the project grid
func renderProjectGrid(w http.ResponseWriter, r *http.Request, trigger string) error {
	projectService := app.GetProjectService()
	projects, err := projectService.List()
	if err != nil {
		slog.Error("Failed to fetch projects for grid render",
			"layer", "handlers",
			"operation", "render_project_grid",
			"trigger", trigger,
			"error", err)
		return err
	}

	projectViews := ConvertProjectsToViews(projects)
	component := projectcomponent.ProjectGrid(projectViews, len(projectViews) > 0)

	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("HX-Trigger-After-Settle", trigger)

	return component.Render(r.Context(), w)
}

// RenderComponent renders a templ component with error handling
func RenderComponent(w http.ResponseWriter, r *http.Request, component templ.Component, operation string) error {
	if err := component.Render(r.Context(), w); err != nil {
		slog.Error("Failed to render component",
			"layer", "handlers",
			"operation", operation,
			"error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return err
	}
	return nil
}

// SetupSSE configures Server-Sent Events headers
func SetupSSE(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
}

// StreamOutput handles the streaming of output to SSE clients
func StreamOutput(w http.ResponseWriter, outputChan <-chan docker.StreamMessage, streamType string) error {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return errors.New("streaming not supported")
	}

	// Send initial connection message
	if _, err := fmt.Fprintf(w, "data: {\"type\":\"info\",\"content\":\"Connected to %s stream\"}\n\n", streamType); err != nil {
		return err
	}
	flusher.Flush()

	// Stream output
	for msg := range outputChan {
		// Convert StreamMessage to JSON
		jsonMsg, err := json.Marshal(msg)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "data: %s\n\n", jsonMsg); err != nil {
			return err
		}
		flusher.Flush()
	}

	// Send completion message
	if _, err := fmt.Fprintf(w, "data: {\"type\":\"success\",\"content\":\"%s finished\"}\n\n", strings.ToUpper(streamType[:1])+streamType[1:]); err != nil {
		return err
	}
	flusher.Flush()

	return nil
}

// LogOperationError logs errors with consistent structure
func LogOperationError(operation, layer string, err error, fields ...any) {
	args := []any{"layer", layer, "operation", operation, "error", err}
	args = append(args, fields...)
	slog.Error("Operation failed", args...)
}

// Middleware functions

// withProjectID middleware extracts and validates project ID, passing it to the next handler
func withProjectID(next func(w http.ResponseWriter, r *http.Request, projectID uuid.UUID)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID, err := ParseProjectID(r)
		if err != nil {
			LogOperationError("parse_project_id", "handlers", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		next(w, r, projectID)
	}
}

// WithFormParsing middleware parses form data before calling the next handler
func WithFormParsing(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			LogOperationError("parse_form", "handlers", err)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		next(w, r)
	}
}

// Generic handler patterns

// HandleModal creates a generic handler for modal endpoints
func HandleModal(modalFunc func(uuid.UUID) (templ.Component, error), operation string) http.HandlerFunc {
	return withProjectID(func(w http.ResponseWriter, r *http.Request, projectID uuid.UUID) {
		component, err := modalFunc(projectID)
		if err != nil {
			LogOperationError(operation, "handlers", err, "project_id", projectID)
			http.Error(w, "Project not found", http.StatusNotFound)
			return
		}
		if err := RenderComponent(w, r, component, operation); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	})
}

// HandleHTMLContent creates a generic handler for HTML content endpoints
func HandleHTMLContent(htmlFunc func(uuid.UUID) (string, error)) http.HandlerFunc {
	return withProjectID(func(w http.ResponseWriter, r *http.Request, projectID uuid.UUID) {
		content, err := htmlFunc(projectID)
		if err != nil {
			LogOperationError("html_content", "handlers", err, "project_id", projectID)
			http.Error(w, fmt.Sprintf("Error: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if _, err := w.Write([]byte(content)); err != nil {
			LogOperationError("html_content_write", "handlers", err, "project_id", projectID)
		}
	})
}

// HandleStream creates a generic handler for streaming endpoints
func HandleStream(streamFunc func(uuid.UUID, chan<- docker.StreamMessage) error, streamType string) http.HandlerFunc {
	return withProjectID(func(w http.ResponseWriter, r *http.Request, projectID uuid.UUID) {
		SetupSSE(w)

		outputChan := make(chan docker.StreamMessage, 100)

		// Start streaming in a goroutine
		go func() {
			defer close(outputChan) // Close channel when streaming function completes
			if err := streamFunc(projectID, outputChan); err != nil {
				LogOperationError(fmt.Sprintf("%s_stream", streamType), "handlers", err, "project_id", projectID)
				// Send error as StreamMessage
				errorMsg := docker.StreamMessage{
					Type: "error",
					Content: fmt.Sprintf(
						"%s failed: %s",
						strings.ToUpper(streamType[:1])+streamType[1:],
						err.Error(),
					),
				}
				select {
				case outputChan <- errorMsg:
				default:
				}
			}
		}()

		if err := StreamOutput(w, outputChan, streamType); err != nil {
			LogOperationError(fmt.Sprintf("%s_stream_output", streamType), "handlers", err, "project_id", projectID)
		}
	})
}

// HandleProjectAction creates a generic handler for project actions (create/update/delete)
func HandleProjectAction(actionFunc func(*http.Request) error, successTrigger, operation string) http.HandlerFunc {
	return WithFormParsing(func(w http.ResponseWriter, r *http.Request) {
		if err := actionFunc(r); err != nil {
			LogOperationError(operation, "handlers", err)
			http.Error(
				w,
				fmt.Sprintf("Failed to %s project: %v", strings.ReplaceAll(operation, "_", " "), err),
				http.StatusInternalServerError,
			)
			return
		}

		if err := renderProjectGrid(w, r, successTrigger); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	})
}
