package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/oar-cd/oar/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseProjectID(t *testing.T) {
	tests := []struct {
		name          string
		projectID     string
		expectedID    uuid.UUID
		expectedError bool
		errorMessage  string
	}{
		{
			name:          "valid UUID",
			projectID:     "123e4567-e89b-12d3-a456-426614174000",
			expectedID:    uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
			expectedError: false,
		},
		{
			name:          "empty project ID",
			projectID:     "",
			expectedError: true,
			errorMessage:  "project ID is required",
		},
		{
			name:          "invalid UUID format",
			projectID:     "invalid-uuid",
			expectedError: true,
			errorMessage:  "invalid project ID format",
		},
		{
			name:          "malformed UUID",
			projectID:     "123e4567-e89b-12d3-a456",
			expectedError: true,
			errorMessage:  "invalid project ID format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a request with chi URL parameter
			req := httptest.NewRequest(http.MethodGet, "/projects/"+tt.projectID, nil)

			// Create chi context and set URL param
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.projectID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			result, err := parseProjectID(req)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Equal(t, uuid.Nil, result)
				assert.Contains(t, err.Error(), tt.errorMessage)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, result)
			}
		})
	}
}

func TestBuildGitAuthConfig(t *testing.T) {
	tests := []struct {
		name       string
		formValues map[string]string
		expected   *services.GitAuthConfig
	}{
		{
			name: "HTTP auth with username and password",
			formValues: map[string]string{
				"auth_method": "http",
				"username":    "testuser",
				"password":    "testpass",
			},
			expected: &services.GitAuthConfig{
				HTTPAuth: &services.GitHTTPAuthConfig{
					Username: "testuser",
					Password: "testpass",
				},
			},
		},
		{
			name: "HTTP auth with only username",
			formValues: map[string]string{
				"auth_method": "http",
				"username":    "testuser",
			},
			expected: &services.GitAuthConfig{
				HTTPAuth: &services.GitHTTPAuthConfig{
					Username: "testuser",
					Password: "",
				},
			},
		},
		{
			name: "HTTP auth with only password",
			formValues: map[string]string{
				"auth_method": "http",
				"password":    "testpass",
			},
			expected: &services.GitAuthConfig{
				HTTPAuth: &services.GitHTTPAuthConfig{
					Username: "",
					Password: "testpass",
				},
			},
		},
		{
			name: "SSH auth with private key and username",
			formValues: map[string]string{
				"auth_method":  "ssh",
				"ssh_username": "git",
				"private_key":  "-----BEGIN OPENSSH PRIVATE KEY-----\ntest\n-----END OPENSSH PRIVATE KEY-----",
			},
			expected: &services.GitAuthConfig{
				SSHAuth: &services.GitSSHAuthConfig{
					User:       "git",
					PrivateKey: "-----BEGIN OPENSSH PRIVATE KEY-----\ntest\n-----END OPENSSH PRIVATE KEY-----",
				},
			},
		},
		{
			name: "SSH auth with only private key",
			formValues: map[string]string{
				"auth_method": "ssh",
				"private_key": "-----BEGIN OPENSSH PRIVATE KEY-----\ntest\n-----END OPENSSH PRIVATE KEY-----",
			},
			expected: &services.GitAuthConfig{
				SSHAuth: &services.GitSSHAuthConfig{
					User:       "",
					PrivateKey: "-----BEGIN OPENSSH PRIVATE KEY-----\ntest\n-----END OPENSSH PRIVATE KEY-----",
				},
			},
		},
		{
			name: "HTTP auth with empty credentials",
			formValues: map[string]string{
				"auth_method": "http",
			},
			expected: nil,
		},
		{
			name: "SSH auth without private key",
			formValues: map[string]string{
				"auth_method":  "ssh",
				"ssh_username": "git",
			},
			expected: nil,
		},
		{
			name: "unknown auth method",
			formValues: map[string]string{
				"auth_method": "token",
				"token":       "github_pat_123",
			},
			expected: nil,
		},
		{
			name: "no auth method",
			formValues: map[string]string{
				"username": "testuser",
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create form data
			formData := url.Values{}
			for key, value := range tt.formValues {
				formData.Set(key, value)
			}

			req := httptest.NewRequest(http.MethodPost, "/projects", strings.NewReader(formData.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			err := req.ParseForm()
			require.NoError(t, err)

			result := buildGitAuthConfig(req)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertProjectToView(t *testing.T) {
	// Test project with all fields
	projectID := uuid.New()
	lastCommit := "abc123"

	project := &services.Project{
		ID:     projectID,
		Name:   "test-project",
		GitURL: "https://github.com/test/repo",
		GitAuth: &services.GitAuthConfig{
			HTTPAuth: &services.GitHTTPAuthConfig{
				Username: "token",
				Password: "github_pat_123",
			},
		},
		Status:       services.ProjectStatusRunning,
		LastCommit:   &lastCommit,
		ComposeFiles: []string{"docker-compose.yml", "docker-compose.prod.yml"},
		Variables:    []string{"ENV=production", "PORT=8080"},
		CreatedAt:    time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		UpdatedAt:    time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC),
	}

	view := convertProjectToView(project)

	assert.Equal(t, projectID, view.ID)
	assert.Equal(t, "test-project", view.Name)
	assert.Equal(t, "https://github.com/test/repo", view.GitURL)
	assert.Equal(t, "running", view.Status)
	assert.Equal(t, &lastCommit, view.LastCommit)
	assert.Equal(t, []string{"docker-compose.yml", "docker-compose.prod.yml"}, view.ComposeFiles)
	assert.Equal(t, []string{"ENV=production", "PORT=8080"}, view.Variables)
	assert.Equal(t, time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC), view.CreatedAt)
	assert.Equal(t, time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC), view.UpdatedAt)

	// Verify GitAuth conversion
	require.NotNil(t, view.GitAuth)
	require.NotNil(t, view.GitAuth.HTTPAuth)
	assert.Equal(t, "token", view.GitAuth.HTTPAuth.Username)
	assert.Equal(t, "github_pat_123", view.GitAuth.HTTPAuth.Password)
	assert.Nil(t, view.GitAuth.SSHAuth)
}

func TestConvertProjectsToViews(t *testing.T) {
	projects := []*services.Project{
		{
			ID:     uuid.New(),
			Name:   "project1",
			GitURL: "https://github.com/test/repo1",
			Status: services.ProjectStatusStopped,
		},
		{
			ID:     uuid.New(),
			Name:   "project2",
			GitURL: "https://github.com/test/repo2",
			Status: services.ProjectStatusRunning,
		},
	}

	views := convertProjectsToViews(projects)

	assert.Len(t, views, 2)
	assert.Equal(t, projects[0].ID, views[0].ID)
	assert.Equal(t, "project1", views[0].Name)
	assert.Equal(t, "stopped", views[0].Status)
	assert.Equal(t, projects[1].ID, views[1].ID)
	assert.Equal(t, "project2", views[1].Name)
	assert.Equal(t, "running", views[1].Status)
}

func TestConvertGitAuthConfig(t *testing.T) {
	tests := []struct {
		name     string
		input    *services.GitAuthConfig
		expected interface{}
	}{
		{
			name:     "nil auth config",
			input:    nil,
			expected: (*interface{})(nil),
		},
		{
			name: "HTTP auth only",
			input: &services.GitAuthConfig{
				HTTPAuth: &services.GitHTTPAuthConfig{
					Username: "testuser",
					Password: "testpass",
				},
			},
			expected: "http",
		},
		{
			name: "SSH auth only",
			input: &services.GitAuthConfig{
				SSHAuth: &services.GitSSHAuthConfig{
					User:       "git",
					PrivateKey: "private-key-content",
				},
			},
			expected: "ssh",
		},
		{
			name: "both auth methods (should convert both)",
			input: &services.GitAuthConfig{
				HTTPAuth: &services.GitHTTPAuthConfig{
					Username: "testuser",
					Password: "testpass",
				},
				SSHAuth: &services.GitSSHAuthConfig{
					User:       "git",
					PrivateKey: "private-key-content",
				},
			},
			expected: "both",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertGitAuthConfig(tt.input)

			switch tt.expected {
			case nil:
				assert.Nil(t, result)
			case "http":
				require.NotNil(t, result)
				require.NotNil(t, result.HTTPAuth)
				assert.Equal(t, "testuser", result.HTTPAuth.Username)
				assert.Equal(t, "testpass", result.HTTPAuth.Password)
				assert.Nil(t, result.SSHAuth)
			case "ssh":
				require.NotNil(t, result)
				assert.Nil(t, result.HTTPAuth)
				require.NotNil(t, result.SSHAuth)
				assert.Equal(t, "git", result.SSHAuth.User)
				assert.Equal(t, "private-key-content", result.SSHAuth.PrivateKey)
			case "both":
				require.NotNil(t, result)
				require.NotNil(t, result.HTTPAuth)
				assert.Equal(t, "testuser", result.HTTPAuth.Username)
				assert.Equal(t, "testpass", result.HTTPAuth.Password)
				require.NotNil(t, result.SSHAuth)
				assert.Equal(t, "git", result.SSHAuth.User)
				assert.Equal(t, "private-key-content", result.SSHAuth.PrivateKey)
			}
		})
	}
}
