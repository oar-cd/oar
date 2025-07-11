package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/ch00k/oar/internal/app"
	"github.com/ch00k/oar/services"
	"github.com/ch00k/oar/ui/pages"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

func main() {
	// Initialize application
	dataDir := os.Getenv("OAR_DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}

	if err := app.Initialize(dataDir); err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)

	// Serve static files
	r.Handle("/assets/*", http.StripPrefix("/assets/", http.FileServer(http.Dir("./ui/assets/"))))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		projectService := app.GetProjectService()
		projects, err := projectService.ListProjects()
		if err != nil {
			log.Printf("Failed to list projects: %v", err)
			projects = []*services.Project{} // Empty slice on error
		}

		component := pages.Home(projects)
		if err := component.Render(r.Context(), w); err != nil {
			http.Error(w, "Failed to render home page", http.StatusInternalServerError)
		}
	})

	r.Post("/projects/{projectID}/edit", func(w http.ResponseWriter, r *http.Request) {
		projectService := app.GetProjectService()

		// Parse project ID from URL
		projectIDStr := chi.URLParam(r, "projectID")
		projectID, err := uuid.Parse(projectIDStr)
		if err != nil {
			http.Error(w, "Invalid project ID", http.StatusBadRequest)
			return
		}

		// Parse form data
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}

		// Parse status
		status, _ := services.ParseProjectStatus(r.FormValue("status"))

		// Parse arrays from form values (tagsinput sends multiple values with same name)
		composeFiles := r.Form["compose_files"]
		environmentFiles := r.Form["environment_files"]

		// Handle last commit (optional)
		var lastCommit *string
		if lc := r.FormValue("last_commit"); lc != "" {
			lastCommit = &lc
		}

		// Create complete project struct
		project := &services.Project{
			ID:               projectID,
			Name:             r.FormValue("name"),
			GitURL:           r.FormValue("git_url"),
			WorkingDir:       r.FormValue("working_dir"),
			ComposeFiles:     composeFiles,
			EnvironmentFiles: environmentFiles,
			Status:           status,
			LastCommit:       lastCommit,
		}

		// Update project
		updatedProject, err := projectService.UpdateProject(project)
		if err != nil {
			log.Printf("Failed to update project: %v", err)
			// Add custom header for JavaScript to detect failed update
			w.Header().Set("HX-Trigger", fmt.Sprintf("project-updated-%s", projectID))

			// Return updated project card HTML with error toast
			component := pages.ProjectCardWithErrorToast(
				project, // Return original project on error
				"Failed to update project",
				"There was an error updating the project. Please try again.",
			)
			if err := component.Render(r.Context(), w); err != nil {
				http.Error(w, "Failed to render project card", http.StatusInternalServerError)
				return
			}
		}

		// Add custom header for JavaScript to detect successful update
		w.Header().Set("HX-Trigger", fmt.Sprintf("project-updated-%s", projectID))

		// Return updated project card HTML with success toast
		component := pages.ProjectCardWithSuccessToast(
			updatedProject,
			"Project updated successfully",
			"Project has been updated with your changes.",
		)
		if err := component.Render(r.Context(), w); err != nil {
			http.Error(w, "Failed to render project card", http.StatusInternalServerError)
		}
	})

	r.Post("/projects/create", func(w http.ResponseWriter, r *http.Request) {
		projectService := app.GetProjectService()

		// Parse form data
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}

		name := r.FormValue("name")
		gitURL := r.FormValue("git_url")

		// Parse compose files and environment files arrays
		composeFiles := r.Form["compose_files"]
		environmentFiles := r.Form["environment_files"]

		// Create new project
		project := services.NewProject(name, gitURL, composeFiles, environmentFiles)

		// Create project via service
		createdProject, err := projectService.CreateProject(&project)
		if err != nil {
			log.Printf("Failed to create project: %v", err)
			// Add custom header for JavaScript to detect failed creation
			w.Header().Set("HX-Trigger", "project-created")

			// Get all projects to refresh the grid
			projects, err := projectService.ListProjects()
			if err != nil {
				log.Printf("Failed to list projects after creation failure: %v", err)
				projects = []*services.Project{} // Empty list on error
			}

			// Return updated projects grid with error toast
			component := pages.ProjectsGridWithErrorToast(
				projects,
				"Failed to create project",
				"There was an error creating the project. Please check your inputs and try again.",
			)
			if err := component.Render(r.Context(), w); err != nil {
				http.Error(w, "Failed to render projects grid", http.StatusInternalServerError)
			}
			return
		}

		// Add custom header to trigger modal close
		w.Header().Set("HX-Trigger", "project-created")

		// Get all projects to refresh the grid
		projects, err := projectService.ListProjects()
		if err != nil {
			log.Printf("Failed to list projects after creation: %v", err)
			projects = []*services.Project{createdProject} // Fallback to just the new project
		}

		// Return updated projects grid with success toast
		component := pages.ProjectsGridWithSuccessToast(
			projects,
			"Project created successfully",
			"New project has been added and cloned.",
		)
		if err := component.Render(r.Context(), w); err != nil {
			http.Error(w, "Failed to render projects grid", http.StatusInternalServerError)
		}
	})

	r.Delete("/projects/{projectID}", func(w http.ResponseWriter, r *http.Request) {
		projectService := app.GetProjectService()

		// Parse project ID from URL
		projectIDStr := chi.URLParam(r, "projectID")
		projectID, err := uuid.Parse(projectIDStr)
		if err != nil {
			http.Error(w, "Invalid project ID", http.StatusBadRequest)
			return
		}

		// Delete project
		err = projectService.RemoveProject(projectID)
		if err != nil {
			log.Printf("Failed to delete project: %v", err)
			// Add custom header for JavaScript to detect failed deletion
			w.Header().Set("HX-Trigger", "project-deleted")

			// Get all projects to refresh the grid
			projects, err := projectService.ListProjects()
			if err != nil {
				log.Printf("Failed to list projects after deletion failure: %v", err)
				projects = []*services.Project{} // Empty list on error
			}

			// Return updated projects grid with error toast
			component := pages.ProjectsGridWithErrorToast(
				projects,
				"Failed to delete project",
				"There was an error deleting the project. Please try again.",
			)
			if err := component.Render(r.Context(), w); err != nil {
				http.Error(w, "Failed to render projects grid", http.StatusInternalServerError)
			}
			return
		}

		// Add custom header to trigger modal close
		w.Header().Set("HX-Trigger", "project-deleted")

		// Get all projects to refresh the grid
		projects, err := projectService.ListProjects()
		if err != nil {
			log.Printf("Failed to list projects after deletion: %v", err)
			projects = []*services.Project{} // Empty list on error
		}

		// Return updated projects grid with success toast
		component := pages.ProjectsGridWithSuccessToast(
			projects,
			"Project deleted successfully",
			"Project has been removed and all data cleaned up.",
		)
		if err := component.Render(r.Context(), w); err != nil {
			http.Error(w, "Failed to render projects grid", http.StatusInternalServerError)
		}
	})

	log.Printf("Server starting on http://127.0.0.1:3333")
	if err := http.ListenAndServe("127.0.0.1:3333", r); err != nil {
		panic(err)
	}
}
