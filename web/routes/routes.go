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
			component := pages.Home(handlers.GetVersion())
			if err := handlers.RenderComponent(w, r, component, "home_page_fallback"); err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		projectViews := handlers.ConvertProjectsToViews(projects)

		var component templ.Component
		if len(projectViews) > 0 {
			component = pages.HomeWithProjects(projectViews, handlers.GetVersion())
		} else {
			component = pages.Home(handlers.GetVersion())
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
			r.Get("/config", handlers.HandleModal(getConfigProjectModalWithLoading, "config_project_modal"))
			r.Get("/config/content", handlers.HandleHTMLContent(getConfigProjectContent))
			r.Get("/deploy", handlers.HandleModal(getDeployProjectModal, "deploy_project_modal"))
			r.Get("/stop", handlers.HandleModal(getStopProjectModal, "stop_project_modal"))
			r.Get("/logs", handlers.HandleModal(getLogsProjectModalWithLoading, "logs_project_modal"))
			r.Get("/logs/content", handlers.HandleHTMLContent(getLogsProjectContent))
			r.Get("/deployments", handlers.HandleModal(getDeploymentsProjectModal, "deployments_project_modal"))

			// Streaming endpoints
			r.Post("/deploy/stream", handlers.HandleStream(actions.DeployProject, "deployment"))
			r.Post("/stop/stream", handlers.HandleStream(actions.StopProject, "stop"))

			// Status pill updates
			r.Get("/status", handlers.HandleModal(getProjectStatusPill, "project_status_pill"))
		})
	})
}

// RegisterUtilityRoutes registers utility routes like auth testing
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

// New functions for loading modals and content-only endpoints
func getConfigProjectModalWithLoading(projectID uuid.UUID) (templ.Component, error) {
	projectService := app.GetProjectService()
	targetProject, err := projectService.Get(projectID)
	if err != nil {
		return nil, err
	}

	projectView := handlers.ConvertProjectToView(targetProject)
	return modals.ConfigProjectModalWithLoading(projectView), nil
}

// formatStdoutStderr formats stdout and stderr with proper styling and parsing
func formatStdoutStderr(stdout, stderr string) string {
	var content strings.Builder

	// Add stderr warnings if any (in orange, with log parsing)
	if stderr != "" {
		content.WriteString(`<span class="deploy-text-stderr">`)
		// Parse each line of stderr to extract clean messages
		lines := strings.Split(stderr, "\n")
		for i, line := range lines {
			if i > 0 {
				content.WriteString("\n")
			}
			parsedLine := services.ParseComposeLogLine(line)
			content.WriteString(parsedLine)
		}
		content.WriteString("</span>\n")
	}

	// Add main stdout (in normal color)
	if stdout != "" {
		content.WriteString(`<span class="deploy-text-stdout">`)
		content.WriteString(stdout)
		content.WriteString("</span>")
	}

	return content.String()
}

func getConfigProjectContent(projectID uuid.UUID) (string, error) {
	projectService := app.GetProjectService()
	config, stderr, err := projectService.GetConfig(projectID)
	if err != nil {
		return fmt.Sprintf(`<pre id="config-content" class="streaming-output">Error getting project configuration:

%s</pre>`, err.Error()), nil
	}

	formattedContent := formatStdoutStderr(config, stderr)
	return fmt.Sprintf(`<pre id="config-content" class="streaming-output">%s</pre>`, formattedContent), nil
}

func getLogsProjectModalWithLoading(projectID uuid.UUID) (templ.Component, error) {
	projectService := app.GetProjectService()
	targetProject, err := projectService.Get(projectID)
	if err != nil {
		return nil, err
	}

	projectView := handlers.ConvertProjectToView(targetProject)
	return modals.LogsProjectModalWithLoading(projectView), nil
}

func getLogsProjectContent(projectID uuid.UUID) (string, error) {
	projectService := app.GetProjectService()
	stdout, stderr, err := projectService.GetLogs(projectID)
	if err != nil {
		return fmt.Sprintf(`<pre id="static-logs-content" class="streaming-output">Error getting project logs:

%s</pre><script>
// Scroll to bottom after content is loaded
const logsOutput = document.getElementById('logs-output');
if (logsOutput) {
	logsOutput.scrollTop = logsOutput.scrollHeight;
}
</script>`, err.Error()), nil
	}

	// Check if stdout is empty - if so, show "No logs available"
	if len(strings.TrimSpace(stdout)) == 0 {
		stdout = "No logs available"
	}

	formattedContent := formatStdoutStderr(stdout, stderr)
	return fmt.Sprintf(`<pre id="static-logs-content" class="streaming-output">%s</pre><script>
// Scroll to bottom after content is loaded
const logsOutput = document.getElementById('logs-output');
if (logsOutput) {
	logsOutput.scrollTop = logsOutput.scrollHeight;
}
</script>`, formattedContent), nil
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
