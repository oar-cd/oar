package main

import (
	"testing"

	"github.com/ch00k/oar/services"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestValidateProjectCreateRequest(t *testing.T) {
	tests := []struct {
		name        string
		req         *ProjectCreateRequest
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid request",
			req: &ProjectCreateRequest{
				Name:         "test-project",
				GitURL:       "https://github.com/test/repo",
				ComposeFiles: "docker-compose.yml",
				Variables:    "ENV=prod",
			},
			expectError: false,
		},
		{
			name: "missing name",
			req: &ProjectCreateRequest{
				GitURL:       "https://github.com/test/repo",
				ComposeFiles: "docker-compose.yml",
			},
			expectError: true,
			errorMsg:    "name is required",
		},
		{
			name: "missing git URL",
			req: &ProjectCreateRequest{
				Name:         "test-project",
				ComposeFiles: "docker-compose.yml",
			},
			expectError: true,
			errorMsg:    "git URL is required",
		},
		{
			name: "missing compose files",
			req: &ProjectCreateRequest{
				Name:   "test-project",
				GitURL: "https://github.com/test/repo",
			},
			expectError: true,
			errorMsg:    "compose files are required",
		},
		{
			name: "empty compose files",
			req: &ProjectCreateRequest{
				Name:         "test-project",
				GitURL:       "https://github.com/test/repo",
				ComposeFiles: "   ",
			},
			expectError: true,
			errorMsg:    "compose files are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateProjectCreateRequest(tt.req)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateProjectUpdateRequest(t *testing.T) {
	tests := []struct {
		name        string
		req         *ProjectUpdateRequest
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid request",
			req: &ProjectUpdateRequest{
				ID:           uuid.New(),
				Name:         "updated-project",
				ComposeFiles: "docker-compose.yml",
				Variables:    "ENV=prod",
			},
			expectError: false,
		},
		{
			name: "missing name",
			req: &ProjectUpdateRequest{
				ID:           uuid.New(),
				ComposeFiles: "docker-compose.yml",
			},
			expectError: true,
			errorMsg:    "name is required",
		},
		{
			name: "missing compose files",
			req: &ProjectUpdateRequest{
				ID:   uuid.New(),
				Name: "updated-project",
			},
			expectError: true,
			errorMsg:    "compose files are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateProjectUpdateRequest(tt.req)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseComposeFiles(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "single file",
			input:    "docker-compose.yml",
			expected: []string{"docker-compose.yml"},
		},
		{
			name:     "multiple files",
			input:    "docker-compose.yml\ndocker-compose.prod.yml",
			expected: []string{"docker-compose.yml", "docker-compose.prod.yml"},
		},
		{
			name:     "files with whitespace",
			input:    "  docker-compose.yml\ndocker-compose.prod.yml  \n",
			expected: []string{"docker-compose.yml", "docker-compose.prod.yml"},
		},
		{
			name:     "files with extra newlines",
			input:    "docker-compose.yml\n\ndocker-compose.prod.yml\n",
			expected: []string{"docker-compose.yml", "", "docker-compose.prod.yml"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseComposeFiles(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseVariables(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "single variable",
			input:    "ENV=production",
			expected: []string{"ENV=production"},
		},
		{
			name:     "multiple variables",
			input:    "ENV=production\nPORT=8080\nDEBUG=false",
			expected: []string{"ENV=production", "PORT=8080", "DEBUG=false"},
		},
		{
			name:     "variables with whitespace",
			input:    "  ENV=production  \nPORT=8080\n",
			expected: []string{"ENV=production  ", "PORT=8080"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseVariables(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildProjectFromCreateRequest(t *testing.T) {
	req := &ProjectCreateRequest{
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
	}

	project := buildProjectFromCreateRequest(req)

	assert.Equal(t, "test-project", project.Name)
	assert.Equal(t, "https://github.com/test/repo", project.GitURL)
	assert.Equal(t, []string{"docker-compose.yml", "docker-compose.prod.yml"}, project.ComposeFiles)
	assert.Equal(t, []string{"ENV=production", "PORT=8080"}, project.Variables)
	assert.Equal(t, services.ProjectStatusStopped, project.Status)
	assert.NotEqual(t, uuid.Nil, project.ID)

	// Verify GitAuth is properly copied
	assert.NotNil(t, project.GitAuth)
	assert.NotNil(t, project.GitAuth.HTTPAuth)
	assert.Equal(t, "token", project.GitAuth.HTTPAuth.Username)
	assert.Equal(t, "github_pat_123", project.GitAuth.HTTPAuth.Password)
}

func TestBuildProjectFromCreateRequest_EmptyFields(t *testing.T) {
	req := &ProjectCreateRequest{
		Name:   "test-project",
		GitURL: "https://github.com/test/repo",
		// ComposeFiles and Variables are empty
	}

	project := buildProjectFromCreateRequest(req)

	assert.Equal(t, "test-project", project.Name)
	assert.Equal(t, "https://github.com/test/repo", project.GitURL)
	assert.Nil(t, project.ComposeFiles)
	assert.Nil(t, project.Variables)
	assert.Nil(t, project.GitAuth)
}

func TestApplyProjectUpdateRequest(t *testing.T) {
	// Create original project
	originalProject := &services.Project{
		ID:           uuid.New(),
		Name:         "original-name",
		GitURL:       "https://github.com/original/repo",
		ComposeFiles: []string{"old-compose.yml"},
		Variables:    []string{"OLD=value"},
		Status:       services.ProjectStatusRunning,
	}

	// Create update request
	req := &ProjectUpdateRequest{
		ID:           originalProject.ID, // Should not change the ID
		Name:         "updated-name",
		ComposeFiles: "new-compose.yml\nnew-compose.prod.yml",
		Variables:    "NEW=value\nANOTHER=setting",
		GitAuth: &services.GitAuthConfig{
			SSHAuth: &services.GitSSHAuthConfig{
				User:       "git",
				PrivateKey: "ssh-private-key",
			},
		},
	}

	applyProjectUpdateRequest(originalProject, req)

	// Verify updates were applied
	assert.Equal(t, "updated-name", originalProject.Name)
	assert.Equal(t, []string{"new-compose.yml", "new-compose.prod.yml"}, originalProject.ComposeFiles)
	assert.Equal(t, []string{"NEW=value", "ANOTHER=setting"}, originalProject.Variables)

	// Verify GitAuth was updated
	assert.NotNil(t, originalProject.GitAuth)
	assert.NotNil(t, originalProject.GitAuth.SSHAuth)
	assert.Equal(t, "git", originalProject.GitAuth.SSHAuth.User)
	assert.Equal(t, "ssh-private-key", originalProject.GitAuth.SSHAuth.PrivateKey)

	// Verify fields that should NOT change
	assert.Equal(t, "https://github.com/original/repo", originalProject.GitURL) // GitURL should not change
	assert.Equal(t, services.ProjectStatusRunning, originalProject.Status)      // Status should not change
}

func TestApplyProjectUpdateRequest_EmptyFields(t *testing.T) {
	originalProject := &services.Project{
		Name:         "original-name",
		ComposeFiles: []string{"old-compose.yml"},
		Variables:    []string{"OLD=value"},
		GitAuth: &services.GitAuthConfig{
			HTTPAuth: &services.GitHTTPAuthConfig{
				Username: "old-user",
				Password: "old-pass",
			},
		},
	}

	req := &ProjectUpdateRequest{
		Name: "updated-name",
		// ComposeFiles, Variables, GitAuth are empty/nil
	}

	applyProjectUpdateRequest(originalProject, req)

	assert.Equal(t, "updated-name", originalProject.Name)
	assert.Nil(t, originalProject.ComposeFiles)
	assert.Nil(t, originalProject.Variables)
	assert.Nil(t, originalProject.GitAuth)
}
