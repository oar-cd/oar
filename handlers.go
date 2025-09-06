package main

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	"github.com/ch00k/oar/frontend/components/project"
	"github.com/ch00k/oar/internal/app"
	"github.com/ch00k/oar/services"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Helper functions for common operations

// parseProjectID extracts and validates project ID from URL parameters
func parseProjectID(r *http.Request) (uuid.UUID, error) {
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

// buildGitAuthConfig creates GitAuthConfig from form values
func buildGitAuthConfig(r *http.Request) *services.GitAuthConfig {
	authMethod := r.FormValue("auth_method")

	switch authMethod {
	case "http":
		username := r.FormValue("username")
		password := r.FormValue("password")
		if username != "" || password != "" {
			return &services.GitAuthConfig{
				HTTPAuth: &services.GitHTTPAuthConfig{
					Username: username,
					Password: password,
				},
			}
		}
	case "ssh":
		sshUsername := r.FormValue("ssh_username")
		privateKey := r.FormValue("private_key")
		if privateKey != "" {
			return &services.GitAuthConfig{
				SSHAuth: &services.GitSSHAuthConfig{
					PrivateKey: privateKey,
					User:       sshUsername,
				},
			}
		}
	}

	return nil
}

// convertProjectToView converts a backend Project to frontend ProjectView
func convertProjectToView(p *services.Project) project.ProjectView {
	return project.ProjectView{
		ID:           p.ID,
		Name:         p.Name,
		GitURL:       p.GitURL,
		GitAuth:      convertGitAuthConfig(p.GitAuth),
		Status:       p.Status.String(),
		LastCommit:   p.LastCommit,
		ComposeFiles: p.ComposeFiles,
		Variables:    p.Variables,
		CreatedAt:    p.CreatedAt,
		UpdatedAt:    p.UpdatedAt,
	}
}

// convertProjectsToViews converts backend projects to frontend ProjectView
func convertProjectsToViews(projects []*services.Project) []project.ProjectView {
	views := make([]project.ProjectView, len(projects))
	for i, p := range projects {
		views[i] = convertProjectToView(p)
	}
	return views
}

// convertGitAuthConfig converts backend GitAuthConfig to frontend GitAuthConfig
func convertGitAuthConfig(auth *services.GitAuthConfig) *project.GitAuthConfig {
	if auth == nil {
		return nil
	}

	result := &project.GitAuthConfig{}

	if auth.HTTPAuth != nil {
		result.HTTPAuth = &project.GitHTTPAuthConfig{
			Username: auth.HTTPAuth.Username,
			Password: auth.HTTPAuth.Password,
		}
	}

	if auth.SSHAuth != nil {
		result.SSHAuth = &project.GitSSHAuthConfig{
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

	projectViews := convertProjectsToViews(projects)
	component := project.ProjectGrid(projectViews, len(projectViews) > 0)

	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("HX-Trigger-After-Settle", trigger)

	return component.Render(r.Context(), w)
}

// renderComponent renders a templ component with error handling
func renderComponent(w http.ResponseWriter, r *http.Request, component templ.Component, operation string) error {
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

// setupSSE configures Server-Sent Events headers
func setupSSE(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
}

// streamOutput handles the streaming of output to SSE clients
func streamOutput(w http.ResponseWriter, outputChan <-chan string, streamType string) error {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return errors.New("streaming not supported")
	}

	// Send initial connection message
	if _, err := fmt.Fprintf(w, "data: {\"type\":\"status\",\"message\":\"Connected to %s stream\"}\n\n", streamType); err != nil {
		return err
	}
	flusher.Flush()

	// Stream output
	for output := range outputChan {
		// Output is already JSON from the source (compose.go or project.go)
		if _, err := fmt.Fprintf(w, "data: %s\n\n", output); err != nil {
			return err
		}
		flusher.Flush()
	}

	// Send completion message
	if _, err := fmt.Fprintf(w, "data: {\"type\":\"complete\",\"message\":\"%s finished\"}\n\n", strings.ToUpper(streamType[:1])+streamType[1:]); err != nil {
		return err
	}
	flusher.Flush()

	return nil
}

// logOperationError logs errors with consistent structure
func logOperationError(operation, layer string, err error, fields ...any) {
	args := []any{"layer", layer, "operation", operation, "error", err}
	args = append(args, fields...)
	slog.Error("Operation failed", args...)
}

// Middleware functions

// withProjectID middleware extracts and validates project ID, passing it to the next handler
func withProjectID(next func(w http.ResponseWriter, r *http.Request, projectID uuid.UUID)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID, err := parseProjectID(r)
		if err != nil {
			logOperationError("parse_project_id", "handlers", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		next(w, r, projectID)
	}
}

// withFormParsing middleware parses form data before calling the next handler
func withFormParsing(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			logOperationError("parse_form", "handlers", err)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		next(w, r)
	}
}

// Generic handler patterns

// handleModal creates a generic handler for modal endpoints
func handleModal(modalFunc func(uuid.UUID) (templ.Component, error), operation string) http.HandlerFunc {
	return withProjectID(func(w http.ResponseWriter, r *http.Request, projectID uuid.UUID) {
		component, err := modalFunc(projectID)
		if err != nil {
			logOperationError(operation, "handlers", err, "project_id", projectID)
			http.Error(w, "Project not found", http.StatusNotFound)
			return
		}
		if err := renderComponent(w, r, component, operation); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	})
}

// handleStream creates a generic handler for streaming endpoints
func handleStream(streamFunc func(uuid.UUID, chan<- string) error, streamType string) http.HandlerFunc {
	return withProjectID(func(w http.ResponseWriter, r *http.Request, projectID uuid.UUID) {
		setupSSE(w)

		outputChan := make(chan string, 100)

		// Start streaming in a goroutine
		go func() {
			defer close(outputChan) // Close channel when streaming function completes
			if err := streamFunc(projectID, outputChan); err != nil {
				logOperationError(fmt.Sprintf("%s_stream", streamType), "handlers", err, "project_id", projectID)
				select {
				case outputChan <- fmt.Sprintf("ERROR: %s failed: %s", strings.ToUpper(streamType[:1])+streamType[1:], err.Error()):
				default:
				}
			}
		}()

		if err := streamOutput(w, outputChan, streamType); err != nil {
			logOperationError(fmt.Sprintf("%s_stream_output", streamType), "handlers", err, "project_id", projectID)
		}
	})
}

// handleProjectAction creates a generic handler for project actions (create/update/delete)
func handleProjectAction(actionFunc func(*http.Request) error, successTrigger, operation string) http.HandlerFunc {
	return withFormParsing(func(w http.ResponseWriter, r *http.Request) {
		if err := actionFunc(r); err != nil {
			logOperationError(operation, "handlers", err)
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
