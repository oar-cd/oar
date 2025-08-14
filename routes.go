package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	"github.com/ch00k/oar/frontend/components/modals"
	"github.com/ch00k/oar/frontend/components/project"
	"github.com/ch00k/oar/frontend/pages"
	"github.com/ch00k/oar/internal/app"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Route registration functions

// registerHomeRoutes registers the home page route
func registerHomeRoutes(r chi.Router) {
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		projectService := app.GetProjectService()
		projects, err := projectService.List()
		if err != nil {
			logOperationError("list_projects", "main", err)
			// Fall back to empty state on error
			component := pages.Home(GetServerVersion())
			if err := renderComponent(w, r, component, "home_page_fallback"); err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		projectViews := convertProjectsToViews(projects)

		var component templ.Component
		if len(projectViews) > 0 {
			component = pages.HomeWithProjects(projectViews, GetServerVersion())
		} else {
			component = pages.Home(GetServerVersion())
		}

		if err := renderComponent(w, r, component, "home_page"); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	})
}

// registerProjectRoutes registers all project-related routes
func registerProjectRoutes(r chi.Router) {
	r.Route("/projects", func(r chi.Router) {
		// Project creation
		r.Get("/create", func(w http.ResponseWriter, r *http.Request) {
			component := modals.CreateProjectModal()
			if err := renderComponent(w, r, component, "create_project_modal"); err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		})
		r.Post("/create", handleProjectAction(createProject, "projectCreated", "create_project"))

		// Individual project routes
		r.Route("/{id}", func(r chi.Router) {
			// Project management
			r.Get("/edit", handleModal(getEditProjectModal, "edit_project_modal"))
			r.Post("/edit", handleProjectAction(updateProject, "projectUpdated", "update_project"))
			r.Get("/delete", handleModal(getDeleteProjectModal, "delete_project_modal"))
			r.Delete("/", handleProjectAction(deleteProject, "projectDeleted", "delete_project"))

			// Project actions
			r.Get("/config", handleModal(getConfigProjectModal, "config_project_modal"))
			r.Get("/deploy", handleModal(getDeployProjectModal, "deploy_project_modal"))
			r.Get("/stop", handleModal(getStopProjectModal, "stop_project_modal"))
			r.Get("/logs", handleModal(getLogsProjectModal, "logs_project_modal"))

			// Streaming endpoints
			r.Post("/deploy/stream", handleStream(deployProject, "deployment"))
			r.Post("/stop/stream", handleStream(stopProject, "stop"))
			r.Post("/logs/stream", handleStream(getProjectLogs, "logs"))

			// Status pill updates
			r.Get("/status", handleModal(getProjectStatusPill, "project_status_pill"))
		})
	})
}

// registerUtilityRoutes registers utility routes like auth testing and discovery
func registerUtilityRoutes(r chi.Router) {
	// Test git authentication
	r.Post("/test-git-auth", withFormParsing(func(w http.ResponseWriter, r *http.Request) {
		gitURL := r.FormValue("git_url")
		if gitURL == "" {
			w.Header().Set("Content-Type", "text/html")
			w.Header().Set("HX-Trigger-After-Settle", "testAuthError")
			w.WriteHeader(http.StatusOK)
			return
		}

		gitService := app.GetGitService()
		gitAuthConfig := buildGitAuthConfig(r)

		if err := gitService.TestAuthentication(gitURL, gitAuthConfig); err != nil {
			logOperationError("test_git_auth", "main", err, "git_url", gitURL)
			w.Header().Set("Content-Type", "text/html")
			w.Header().Set("HX-Trigger-After-Settle", "testAuthError")
			w.WriteHeader(http.StatusOK)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("HX-Trigger-After-Settle", "testAuthSuccess")
		w.WriteHeader(http.StatusOK)
	}))

	// Discover compose files
	r.Post("/discover", withFormParsing(func(w http.ResponseWriter, r *http.Request) {
		gitURL := r.FormValue("git_url")
		if gitURL == "" {
			w.Header().Set("Content-Type", "text/html")
			w.Header().Set("HX-Trigger-After-Settle", "discoverError")
			w.WriteHeader(http.StatusOK)
			return
		}

		discoveryService := app.GetDiscoveryService()
		gitAuthConfig := buildGitAuthConfig(r)

		discoveryResponse, err := discoveryService.DiscoverFiles(gitURL, gitAuthConfig)
		if err != nil {
			logOperationError("discover", "main", err, "git_url", gitURL)
			w.Header().Set("Content-Type", "text/html")
			w.Header().Set("HX-Trigger-After-Settle", "discoverError")
			w.WriteHeader(http.StatusOK)
			return
		}

		var composeFilePaths []string
		for _, composeFile := range discoveryResponse.ComposeFiles {
			composeFilePaths = append(composeFilePaths, composeFile.Path)
		}

		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("HX-Trigger-After-Settle", "discoverSuccess")
		if _, err := fmt.Fprintf(w, `<textarea
			id="compose_files"
			name="compose_files"
			class="form-textarea"
			rows="3"
			placeholder="docker-compose.yml"
		>%s</textarea>`, strings.Join(composeFilePaths, "\n")); err != nil {
			logOperationError("discover_write", "main", err, "git_url", gitURL)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}))

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			logOperationError("health_check", "main", err)
		}
	})
}

// Modal helper functions

func getEditProjectModal(projectID uuid.UUID) (templ.Component, error) {
	projectService := app.GetProjectService()
	targetProject, err := projectService.Get(projectID)
	if err != nil {
		return nil, err
	}

	projectView := convertProjectToView(targetProject)
	return modals.EditProjectModal(projectView), nil
}

func getDeleteProjectModal(projectID uuid.UUID) (templ.Component, error) {
	projectService := app.GetProjectService()
	targetProject, err := projectService.Get(projectID)
	if err != nil {
		return nil, err
	}

	projectView := convertProjectToView(targetProject)
	return modals.DeleteProjectModal(projectView), nil
}

func getConfigProjectModal(projectID uuid.UUID) (templ.Component, error) {
	projectService := app.GetProjectService()
	targetProject, err := projectService.Get(projectID)
	if err != nil {
		return nil, err
	}

	config, err := projectService.GetConfig(projectID)
	if err != nil {
		config = fmt.Sprintf(
			"Error getting project configuration:\n\n%s\n\nNote: The project repository must be deployed first before configuration can be displayed.",
			err.Error(),
		)
	}

	projectView := convertProjectToView(targetProject)
	return modals.ConfigProjectModal(projectView, config), nil
}

func getDeployProjectModal(projectID uuid.UUID) (templ.Component, error) {
	projectService := app.GetProjectService()
	targetProject, err := projectService.Get(projectID)
	if err != nil {
		return nil, err
	}

	projectView := convertProjectToView(targetProject)
	return modals.DeployProjectModal(projectView), nil
}

func getStopProjectModal(projectID uuid.UUID) (templ.Component, error) {
	projectService := app.GetProjectService()
	targetProject, err := projectService.Get(projectID)
	if err != nil {
		return nil, err
	}

	projectView := convertProjectToView(targetProject)
	return modals.StopProjectModal(projectView), nil
}

func getLogsProjectModal(projectID uuid.UUID) (templ.Component, error) {
	projectService := app.GetProjectService()
	targetProject, err := projectService.Get(projectID)
	if err != nil {
		return nil, err
	}

	projectView := convertProjectToView(targetProject)
	return modals.LogsProjectModal(projectView), nil
}

func getProjectStatusPill(projectID uuid.UUID) (templ.Component, error) {
	projectService := app.GetProjectService()
	targetProject, err := projectService.Get(projectID)
	if err != nil {
		return nil, err
	}

	projectView := convertProjectToView(targetProject)
	return project.StatusPill(projectView.ID.String(), projectView.Status), nil
}
