package handlers

import (
	"log/slog"
	"net/http"

	"github.com/ch00k/oar/services"
	"github.com/go-chi/chi/v5"
)

// BootstrapHandlers handles bootstrap-related HTTP requests
type BootstrapHandlers struct {
	projectManager services.ProjectManager
}

// NewBootstrapHandlers creates a new BootstrapHandlers instance
func NewBootstrapHandlers(projectManager services.ProjectManager) *BootstrapHandlers {
	return &BootstrapHandlers{
		projectManager: projectManager,
	}
}

// Bootstrap creates a project and immediately deploys it
func (h *BootstrapHandlers) Bootstrap(w http.ResponseWriter, r *http.Request) {
	// Parse form data
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	// Extract form values
	name := r.FormValue("name")
	gitURL := r.FormValue("git_url")
	composeFiles := r.Form["compose_files"]
	environmentFiles := r.Form["environment_files"]

	// Validate required fields
	if name == "" || gitURL == "" {
		http.Error(w, "Missing required fields: name and git_url", http.StatusBadRequest)
		return
	}

	// Create project
	project := services.NewProject(name, gitURL, composeFiles, environmentFiles)
	createdProject, err := h.projectManager.Create(&project)
	if err != nil {
		slog.Error("Failed to create project during bootstrap",
			"error", err,
			"name", name,
			"git_url", gitURL)
		http.Error(w, "Failed to create project", http.StatusInternalServerError)
		return
	}

	// Deploy synchronously - should be fast since containers are already running
	if err := h.projectManager.Deploy(createdProject.ID, true); err != nil {
		slog.Error("Failed to deploy project during bootstrap",
			"error", err,
			"project_id", createdProject.ID,
			"name", name)
		http.Error(w, "Failed to deploy project", http.StatusInternalServerError)
		return
	}

	// Return success response
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("Project created and deployed successfully")); err != nil {
		slog.Error("Failed to write response during bootstrap",
			"error", err,
			"project_id", createdProject.ID,
			"name", name)
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
		return
	}
}

// RegisterRoutes registers all bootstrap-related routes
func (h *BootstrapHandlers) RegisterRoutes(r chi.Router) {
	r.Post("/bootstrap", h.Bootstrap)
}
