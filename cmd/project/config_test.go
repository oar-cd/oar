package project

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/ch00k/oar/internal/app"
	"github.com/ch00k/oar/services"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewCmdProjectConfig(t *testing.T) {
	testProjectID := uuid.New()
	testProject := &services.Project{
		ID:        testProjectID,
		Name:      "test-project",
		GitURL:    "https://github.com/test/project.git",
		Status:    services.ProjectStatusRunning,
		CreatedAt: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
	}
	mockConfig := `version: '3.8'
services:
  web:
    image: nginx:latest
    ports:
      - "80:80"
  db:
    image: postgres:13
    environment:
      POSTGRES_DB: testdb`

	tests := []struct {
		name            string
		args            []string
		mockProject     *services.Project
		mockConfig      string
		mockGetError    error
		mockConfigError error
		expectError     bool
		expectedText    string
	}{
		{
			name:            "config success",
			args:            []string{testProjectID.String()},
			mockProject:     testProject,
			mockConfig:      mockConfig,
			mockGetError:    nil,
			mockConfigError: nil,
			expectError:     false,
			expectedText:    "version: '3.8'",
		},
		{
			name:         "project not found",
			args:         []string{testProjectID.String()},
			mockProject:  nil,
			mockGetError: errors.New("project not found"),
			expectError:  true,
		},
		{
			name:            "config error",
			args:            []string{testProjectID.String()},
			mockProject:     testProject,
			mockGetError:    nil,
			mockConfigError: errors.New("failed to get project configuration"),
			expectError:     true,
		},
		{
			name:        "invalid project ID",
			args:        []string{"invalid-uuid"},
			mockProject: nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock
			mockService := &MockProjectManager{
				GetFunc: func(id uuid.UUID) (*services.Project, error) {
					if tt.mockGetError != nil {
						return nil, tt.mockGetError
					}
					return tt.mockProject, nil
				},
				GetConfigFunc: func(projectID uuid.UUID) (string, error) {
					if tt.mockConfigError != nil {
						return "", tt.mockConfigError
					}
					return tt.mockConfig, nil
				},
			}
			app.SetProjectServiceForTesting(mockService)

			// Create command and capture output
			cmd := NewCmdProjectConfig()
			var stdout, stderr bytes.Buffer
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)
			cmd.SetArgs(tt.args)

			// Execute command
			err := cmd.Execute()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				stdoutStr := stdout.String()

				if tt.expectedText != "" {
					assert.Contains(t, stdoutStr, tt.expectedText)
				}

				if tt.mockProject != nil && tt.mockConfigError == nil {
					// Verify project info is shown
					assert.Contains(t, stdoutStr, tt.mockProject.Name)
					// Verify config content is shown
					assert.Contains(t, stdoutStr, tt.mockConfig)
				}
			}
		})
	}
}

func TestNewCmdProjectConfigCommand(t *testing.T) {
	cmd := NewCmdProjectConfig()

	// Test command configuration
	assert.Equal(t, "config <project-id>", cmd.Use)
	assert.Equal(t, "Show the Docker Compose configuration for a project", cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotNil(t, cmd.RunE)

	// Verify the command can be found by name
	assert.Equal(t, "config", cmd.Name())
}
