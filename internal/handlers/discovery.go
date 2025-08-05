package handlers

import (
	"log/slog"
	"net/http"

	"github.com/ch00k/oar/services"
	"github.com/ch00k/oar/ui/components/discovery"
	"github.com/go-chi/chi/v5"
)

// DiscoveryHandlers handles file discovery requests
type DiscoveryHandlers struct {
	discoveryService *services.ProjectDiscoveryService
}

// NewDiscoveryHandlers creates new discovery handlers
func NewDiscoveryHandlers(discoveryService *services.ProjectDiscoveryService) *DiscoveryHandlers {
	return &DiscoveryHandlers{
		discoveryService: discoveryService,
	}
}

// DiscoverFiles discovers compose files in a Git repository and returns HTML
func (h *DiscoveryHandlers) DiscoverFiles(w http.ResponseWriter, r *http.Request) {
	// Parse form data (HTMX sends form data, not JSON)
	if err := r.ParseForm(); err != nil {
		slog.Error("Handler operation failed",
			"layer", "handler",
			"operation", "discover_files",
			"error", err)
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	gitURL := r.FormValue("git_url")
	if gitURL == "" {
		slog.Error("Handler operation failed",
			"layer", "handler",
			"operation", "discover_files",
			"error", "git_url is required")

		// Render error with toast and preserved form
		component := discovery.DiscoveryErrorWithToast("Git URL is required")
		if err := component.Render(r.Context(), w); err != nil {
			http.Error(w, "Failed to render error", http.StatusInternalServerError)
		}
		return
	}

	// Create temporary auth config from form fields
	authConfig := CreateTempAuthConfig(r)

	// Perform discovery with authentication
	response, err := h.discoveryService.DiscoverFiles(gitURL, authConfig)
	if err != nil {
		slog.Error("Handler operation failed",
			"layer", "handler",
			"operation", "discover_files",
			"git_url", gitURL,
			"error", err)

		// Render error with toast and preserved form
		component := discovery.DiscoveryErrorWithToast(err.Error())
		if err := component.Render(r.Context(), w); err != nil {
			http.Error(w, "Failed to render error", http.StatusInternalServerError)
		}
		return
	}

	// Render discovered files template
	component := discovery.DiscoveredFiles(discovery.DiscoveredFilesProps{
		ComposeFiles:  response.ComposeFiles,
		EnvFiles:      response.EnvFiles,
		TempClonePath: response.TempClonePath,
		GitURL:        gitURL,
	})

	if err := component.Render(r.Context(), w); err != nil {
		slog.Error("Handler operation failed",
			"layer", "handler",
			"operation", "discover_files",
			"error", err)
		http.Error(w, "Failed to render discovered files", http.StatusInternalServerError)
	}
}

// RegisterRoutes registers discovery-related routes
func (h *DiscoveryHandlers) RegisterRoutes(r chi.Router) {
	r.Post("/discover", h.DiscoverFiles)
}
