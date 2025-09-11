package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/oar-cd/oar/services"
	"github.com/stretchr/testify/assert"
)

// Helper function to create HTTP request with form data
func createFormRequest(method, path string, formData map[string]string) *http.Request {
	values := url.Values{}
	for key, value := range formData {
		values.Set(key, value)
	}

	req := httptest.NewRequest(method, path, strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err := req.ParseForm(); err != nil {
		panic(fmt.Sprintf("Failed to parse form in test helper: %v", err))
	}

	return req
}

// Helper function to add project ID to request context
func addProjectIDToRequest(req *http.Request, projectID uuid.UUID) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", projectID.String())
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

// Test the request parsing and validation logic of action functions
// Note: These tests focus on input validation rather than service integration
// to avoid complex mocking of the app.GetProjectService() global function

func TestCreateProjectRequestParsing(t *testing.T) {
	tests := []struct {
		name     string
		formData map[string]string
		expected *ProjectCreateRequest
	}{
		{
			name: "complete form data",
			formData: map[string]string{
				"name":          "test-project",
				"git_url":       "https://github.com/test/repo",
				"compose_files": "docker-compose.yml\ndocker-compose.prod.yml",
				"variables":     "ENV=production\nPORT=8080",
				"auth_method":   "http",
				"username":      "token",
				"password":      "github_pat_123",
			},
			expected: &ProjectCreateRequest{
				Name:         "test-project",
				GitURL:       "https://github.com/test/repo",
				ComposeFiles: "docker-compose.yml\ndocker-compose.prod.yml",
				Variables:    "ENV=production\nPORT=8080",
				GitAuth: &services.GitAuthConfig{
					HTTPAuth: &services.GitHTTPAuthConfig{
						Username: "token",
						Password: "github_pat_123",
					},
				},
			},
		},
		{
			name: "minimal form data",
			formData: map[string]string{
				"name":          "minimal-project",
				"git_url":       "https://github.com/test/minimal",
				"compose_files": "docker-compose.yml",
			},
			expected: &ProjectCreateRequest{
				Name:         "minimal-project",
				GitURL:       "https://github.com/test/minimal",
				ComposeFiles: "docker-compose.yml",
				Variables:    "",
				GitAuth:      nil,
			},
		},
		{
			name: "SSH auth",
			formData: map[string]string{
				"name":          "ssh-project",
				"git_url":       "git@github.com:test/repo.git",
				"compose_files": "docker-compose.yml",
				"auth_method":   "ssh",
				"ssh_username":  "git",
				"private_key":   "-----BEGIN OPENSSH PRIVATE KEY-----\ntest\n-----END OPENSSH PRIVATE KEY-----",
			},
			expected: &ProjectCreateRequest{
				Name:         "ssh-project",
				GitURL:       "git@github.com:test/repo.git",
				ComposeFiles: "docker-compose.yml",
				Variables:    "",
				GitAuth: &services.GitAuthConfig{
					SSHAuth: &services.GitSSHAuthConfig{
						User:       "git",
						PrivateKey: "-----BEGIN OPENSSH PRIVATE KEY-----\ntest\n-----END OPENSSH PRIVATE KEY-----",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := createFormRequest(http.MethodPost, "/projects", tt.formData)

			// Extract form data into request struct (simulating createProject logic)
			result := &ProjectCreateRequest{
				Name:         req.FormValue("name"),
				GitURL:       req.FormValue("git_url"),
				ComposeFiles: req.FormValue("compose_files"),
				Variables:    req.FormValue("variables"),
				GitAuth:      buildGitAuthConfig(req),
			}

			assert.Equal(t, tt.expected.Name, result.Name)
			assert.Equal(t, tt.expected.GitURL, result.GitURL)
			assert.Equal(t, tt.expected.ComposeFiles, result.ComposeFiles)
			assert.Equal(t, tt.expected.Variables, result.Variables)
			assert.Equal(t, tt.expected.GitAuth, result.GitAuth)
		})
	}
}

func TestUpdateProjectRequestParsing(t *testing.T) {
	projectID := uuid.New()

	tests := []struct {
		name     string
		formData map[string]string
		expected *ProjectUpdateRequest
	}{
		{
			name: "complete update data",
			formData: map[string]string{
				"name":          "updated-project",
				"compose_files": "new-compose.yml\nnew-compose.prod.yml",
				"variables":     "NEW=value\nANOTHER=setting",
				"auth_method":   "ssh",
				"ssh_username":  "git",
				"private_key":   "new-ssh-key",
			},
			expected: &ProjectUpdateRequest{
				ID:           projectID,
				Name:         "updated-project",
				ComposeFiles: "new-compose.yml\nnew-compose.prod.yml",
				Variables:    "NEW=value\nANOTHER=setting",
				GitAuth: &services.GitAuthConfig{
					SSHAuth: &services.GitSSHAuthConfig{
						User:       "git",
						PrivateKey: "new-ssh-key",
					},
				},
			},
		},
		{
			name: "minimal update data",
			formData: map[string]string{
				"name":          "updated-name",
				"compose_files": "simple-compose.yml",
			},
			expected: &ProjectUpdateRequest{
				ID:           projectID,
				Name:         "updated-name",
				ComposeFiles: "simple-compose.yml",
				Variables:    "",
				GitAuth:      nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := createFormRequest(http.MethodPut, "/projects/"+projectID.String(), tt.formData)
			req = addProjectIDToRequest(req, projectID)

			// Parse project ID (simulating updateProject logic)
			parsedID, err := parseProjectID(req)
			assert.NoError(t, err)
			assert.Equal(t, projectID, parsedID)

			// Extract form data into request struct
			result := &ProjectUpdateRequest{
				ID:           parsedID,
				Name:         req.FormValue("name"),
				ComposeFiles: req.FormValue("compose_files"),
				Variables:    req.FormValue("variables"),
				GitAuth:      buildGitAuthConfig(req),
			}

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDeleteProjectRequestParsing(t *testing.T) {
	projectID := uuid.New()

	req := httptest.NewRequest(http.MethodDelete, "/projects/"+projectID.String(), nil)
	req = addProjectIDToRequest(req, projectID)

	// Parse project ID (simulating deleteProject logic)
	parsedID, err := parseProjectID(req)
	assert.NoError(t, err)
	assert.Equal(t, projectID, parsedID)
}

func TestCreateProjectValidation(t *testing.T) {
	tests := []struct {
		name        string
		formData    map[string]string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid request",
			formData: map[string]string{
				"name":          "test-project",
				"git_url":       "https://github.com/test/repo",
				"compose_files": "docker-compose.yml",
			},
			expectError: false,
		},
		{
			name: "missing name",
			formData: map[string]string{
				"git_url":       "https://github.com/test/repo",
				"compose_files": "docker-compose.yml",
			},
			expectError: true,
			errorMsg:    "name is required",
		},
		{
			name: "missing git URL",
			formData: map[string]string{
				"name":          "test-project",
				"compose_files": "docker-compose.yml",
			},
			expectError: true,
			errorMsg:    "git URL is required",
		},
		{
			name: "missing compose files",
			formData: map[string]string{
				"name":    "test-project",
				"git_url": "https://github.com/test/repo",
			},
			expectError: true,
			errorMsg:    "compose files are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := createFormRequest(http.MethodPost, "/projects", tt.formData)

			// Create request object and validate (simulating createProject logic)
			createReq := &ProjectCreateRequest{
				Name:         req.FormValue("name"),
				GitURL:       req.FormValue("git_url"),
				ComposeFiles: req.FormValue("compose_files"),
				Variables:    req.FormValue("variables"),
				GitAuth:      buildGitAuthConfig(req),
			}

			err := validateProjectCreateRequest(createReq)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUpdateProjectValidation(t *testing.T) {
	projectID := uuid.New()

	tests := []struct {
		name        string
		formData    map[string]string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid request",
			formData: map[string]string{
				"name":          "updated-project",
				"compose_files": "docker-compose.yml",
			},
			expectError: false,
		},
		{
			name: "missing name",
			formData: map[string]string{
				"compose_files": "docker-compose.yml",
			},
			expectError: true,
			errorMsg:    "name is required",
		},
		{
			name: "missing compose files",
			formData: map[string]string{
				"name": "updated-project",
			},
			expectError: true,
			errorMsg:    "compose files are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := createFormRequest(http.MethodPut, "/projects/"+projectID.String(), tt.formData)

			// Create request object and validate (simulating updateProject logic)
			updateReq := &ProjectUpdateRequest{
				ID:           projectID,
				Name:         req.FormValue("name"),
				ComposeFiles: req.FormValue("compose_files"),
				Variables:    req.FormValue("variables"),
				GitAuth:      buildGitAuthConfig(req),
			}

			err := validateProjectUpdateRequest(updateReq)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
