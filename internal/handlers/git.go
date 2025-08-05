package handlers

import (
	"log/slog"
	"net/http"

	"github.com/ch00k/oar/services"
	"github.com/go-chi/chi/v5"
)

// GitHandlers handles Git-related requests
type GitHandlers struct {
	gitService services.GitExecutor
}

// NewGitHandlers creates new Git handlers
func NewGitHandlers(gitService services.GitExecutor) *GitHandlers {
	return &GitHandlers{
		gitService: gitService,
	}
}

// TestGitAuth tests Git authentication without cloning the repository
func (h *GitHandlers) TestGitAuth(w http.ResponseWriter, r *http.Request) {
	// Parse form data
	if err := r.ParseForm(); err != nil {
		slog.Error("Handler operation failed",
			"layer", "handler",
			"operation", "test_git_auth",
			"error", err)
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	// Get Git URL and authentication config
	gitURL := r.FormValue("git_url")
	authConfig := CreateTempAuthConfig(r)

	if gitURL == "" {
		http.Error(w, "Git URL is required", http.StatusBadRequest)
		return
	}

	// Test git authentication by attempting to perform a minimal Git operation
	err := h.gitService.TestAuthentication(gitURL, authConfig)
	if err != nil {
		slog.Error("Handler operation failed",
			"layer", "handler",
			"operation", "test_git_auth",
			"git_url", gitURL,
			"error", err)

		// Return user-friendly error message
		errorMsg := services.FormatErrorForUser(err)
		w.WriteHeader(http.StatusUnauthorized)
		if _, err := w.Write([]byte("Git authentication failed: " + errorMsg)); err != nil {
			slog.Error("Failed to write response",
				"layer", "handler",
				"operation", "test_git_auth",
				"git_url", gitURL,
				"error", err)
			http.Error(w, "Failed to write response", http.StatusInternalServerError)
		}
		return
	}

	// Success
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("Git authentication successful")); err != nil {
		slog.Error("Failed to write response",
			"layer", "handler",
			"operation", "test_git_auth",
			"git_url", gitURL,
			"error", err)
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
	}
}

// RegisterRoutes registers Git-related routes
func (h *GitHandlers) RegisterRoutes(r chi.Router) {
	r.Post("/test-git-auth", h.TestGitAuth)
}
