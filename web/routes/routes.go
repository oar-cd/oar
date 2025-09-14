// Package routes provides HTTP route registration for the web server.
package routes

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/oar-cd/oar/app"
	"github.com/oar-cd/oar/services"
	"github.com/oar-cd/oar/web/actions"
	"github.com/oar-cd/oar/web/components/modals"
	"github.com/oar-cd/oar/web/components/project"
	"github.com/oar-cd/oar/web/handlers"
	"github.com/oar-cd/oar/web/pages"
)

// Route registration functions

// RegisterHomeRoutes registers the home page route
func RegisterHomeRoutes(r chi.Router) {
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		projectService := app.GetProjectService()
		projects, err := projectService.List()
		if err != nil {
			handlers.LogOperationError("list_projects", "main", err)
			// Fall back to empty state on error
			component := pages.Home(handlers.GetServerVersion())
			if err := handlers.RenderComponent(w, r, component, "home_page_fallback"); err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		projectViews := handlers.ConvertProjectsToViews(projects)

		var component templ.Component
		if len(projectViews) > 0 {
			component = pages.HomeWithProjects(projectViews, handlers.GetServerVersion())
		} else {
			component = pages.Home(handlers.GetServerVersion())
		}

		if err := handlers.RenderComponent(w, r, component, "home_page"); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	})
}

// RegisterProjectRoutes registers all project-related routes
func RegisterProjectRoutes(r chi.Router) {
	r.Route("/projects", func(r chi.Router) {
		// Project creation
		r.Get("/create", func(w http.ResponseWriter, r *http.Request) {
			component := modals.CreateProjectModal()
			if err := handlers.RenderComponent(w, r, component, "create_project_modal"); err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		})
		r.Post("/create", handlers.HandleProjectAction(actions.CreateProject, "projectCreated", "create_project"))

		// Individual project routes
		r.Route("/{id}", func(r chi.Router) {
			// Project management
			r.Get("/edit", handlers.HandleModal(getEditProjectModal, "edit_project_modal"))
			r.Post("/edit", handlers.HandleProjectAction(actions.UpdateProject, "projectUpdated", "update_project"))
			r.Get("/delete", handlers.HandleModal(getDeleteProjectModal, "delete_project_modal"))
			r.Delete("/", handlers.HandleProjectAction(actions.DeleteProject, "projectDeleted", "delete_project"))

			// Project actions
			r.Get("/config", handlers.HandleModal(getConfigProjectModal, "config_project_modal"))
			r.Get("/deploy", handlers.HandleModal(getDeployProjectModal, "deploy_project_modal"))
			r.Get("/stop", handlers.HandleModal(getStopProjectModal, "stop_project_modal"))
			r.Get("/logs", handlers.HandleModal(getLogsProjectModal, "logs_project_modal"))
			r.Get("/deployments", handlers.HandleModal(getDeploymentsProjectModal, "deployments_project_modal"))

			// Streaming endpoints
			r.Post("/deploy/stream", handlers.HandleStream(actions.DeployProject, "deployment"))
			r.Post("/stop/stream", handlers.HandleStream(actions.StopProject, "stop"))
			r.Post("/logs/stream", handlers.HandleStream(actions.GetProjectLogs, "logs"))

			// Status pill updates
			r.Get("/status", handlers.HandleModal(getProjectStatusPill, "project_status_pill"))
		})
	})
}

// RegisterUtilityRoutes registers utility routes like auth testing and discovery
func RegisterUtilityRoutes(r chi.Router) {
	// Test git authentication
	r.Post("/test-git-auth", handlers.WithFormParsing(func(w http.ResponseWriter, r *http.Request) {
		gitURL := r.FormValue("git_url")
		if gitURL == "" {
			w.Header().Set("Content-Type", "text/html")
			w.Header().Set("HX-Trigger-After-Settle", "testAuthError")
			w.WriteHeader(http.StatusOK)
			return
		}

		gitService := app.GetGitService()
		gitAuthConfig := handlers.BuildGitAuthConfig(r)

		if err := gitService.TestAuthentication(gitURL, gitAuthConfig); err != nil {
			handlers.LogOperationError("test_git_auth", "main", err, "git_url", gitURL)
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
	r.Post("/discover", handlers.WithFormParsing(func(w http.ResponseWriter, r *http.Request) {
		gitURL := r.FormValue("git_url")
		if gitURL == "" {
			w.Header().Set("Content-Type", "text/html")
			w.Header().Set("HX-Trigger-After-Settle", "discoverError")
			w.WriteHeader(http.StatusOK)
			// Return the original textarea to prevent it from disappearing
			currentComposeFiles := r.FormValue("compose_files")
			if _, err := fmt.Fprintf(w, `<textarea
				id="compose_files"
				name="compose_files"
				class="form-textarea"
				rows="3"
				placeholder="docker-compose.yml"
				required
			>%s</textarea>`, currentComposeFiles); err != nil {
				handlers.LogOperationError("discover_write_error", "main", err)
			}
			return
		}

		discoveryService := app.GetDiscoveryService()
		gitAuthConfig := handlers.BuildGitAuthConfig(r)
		gitBranch := r.FormValue("git_branch") // Extract git branch from form

		discoveryResponse, err := discoveryService.DiscoverFiles(gitURL, gitBranch, gitAuthConfig)
		if err != nil {
			handlers.LogOperationError("discover", "main", err, "git_url", gitURL)
			w.Header().Set("Content-Type", "text/html")
			w.Header().Set("HX-Trigger-After-Settle", "discoverError")
			w.WriteHeader(http.StatusOK)
			// Return the original textarea to prevent it from disappearing
			currentComposeFiles := r.FormValue("compose_files")
			if _, err := fmt.Fprintf(w, `<textarea
				id="compose_files"
				name="compose_files"
				class="form-textarea"
				rows="3"
				placeholder="docker-compose.yml"
				required
			>%s</textarea>`, currentComposeFiles); err != nil {
				handlers.LogOperationError("discover_write_error", "main", err)
			}
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
			handlers.LogOperationError("discover_write", "main", err, "git_url", gitURL)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}))

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			handlers.LogOperationError("health_check", "main", err)
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

	projectView := handlers.ConvertProjectToView(targetProject)
	return modals.EditProjectModal(projectView), nil
}

func getDeleteProjectModal(projectID uuid.UUID) (templ.Component, error) {
	projectService := app.GetProjectService()
	targetProject, err := projectService.Get(projectID)
	if err != nil {
		return nil, err
	}

	deletedDirPath := services.GetDeletedDirectoryPath(targetProject.WorkingDir)
	projectView := handlers.ConvertProjectToView(targetProject)
	return modals.DeleteProjectModal(projectView, deletedDirPath), nil
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

	projectView := handlers.ConvertProjectToView(targetProject)
	return modals.ConfigProjectModal(projectView, config), nil
}

func getDeployProjectModal(projectID uuid.UUID) (templ.Component, error) {
	projectService := app.GetProjectService()
	targetProject, err := projectService.Get(projectID)
	if err != nil {
		return nil, err
	}

	projectView := handlers.ConvertProjectToView(targetProject)
	return modals.DeployProjectModal(projectView), nil
}

func getStopProjectModal(projectID uuid.UUID) (templ.Component, error) {
	projectService := app.GetProjectService()
	targetProject, err := projectService.Get(projectID)
	if err != nil {
		return nil, err
	}

	projectView := handlers.ConvertProjectToView(targetProject)
	return modals.StopProjectModal(projectView), nil
}

func getLogsProjectModal(projectID uuid.UUID) (templ.Component, error) {
	projectService := app.GetProjectService()
	targetProject, err := projectService.Get(projectID)
	if err != nil {
		return nil, err
	}

	projectView := handlers.ConvertProjectToView(targetProject)
	return modals.LogsProjectModal(projectView), nil
}

func getProjectStatusPill(projectID uuid.UUID) (templ.Component, error) {
	projectService := app.GetProjectService()
	targetProject, err := projectService.Get(projectID)
	if err != nil {
		return nil, err
	}

	projectView := handlers.ConvertProjectToView(targetProject)
	return project.StatusPill(projectView.ID.String(), projectView.Status), nil
}

func getDeploymentsProjectModal(projectID uuid.UUID) (templ.Component, error) {
	projectService := app.GetProjectService()
	targetProject, err := projectService.Get(projectID)
	if err != nil {
		return nil, err
	}

	deployments, err := projectService.ListDeployments(projectID)
	if err != nil {
		return nil, err
	}

	projectView := handlers.ConvertProjectToView(targetProject)
	return modals.DeploymentsProjectModal(projectView, deployments), nil
}
