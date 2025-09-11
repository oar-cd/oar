package project

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/ch00k/oar/services"

	"github.com/ch00k/oar/internal/app"
	"github.com/ch00k/oar/testing/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewCmdProjectList(t *testing.T) {
	tests := []struct {
		name           string
		mockProjects   []*services.Project
		mockError      error
		expectedOutput string
		expectError    bool
	}{
		{
			name: "list projects success",
			mockProjects: []*services.Project{
				{
					ID:        uuid.New(),
					Name:      "test-project-1",
					GitURL:    "https://github.com/test/project1.git",
					CreatedAt: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
				},
				{
					ID:        uuid.New(),
					Name:      "test-project-2",
					GitURL:    "https://github.com/test/project2.git",
					CreatedAt: time.Date(2023, 1, 2, 12, 0, 0, 0, time.UTC),
				},
			},
			mockError:   nil,
			expectError: false,
		},
		{
			name:           "no projects found",
			mockProjects:   []*services.Project{},
			mockError:      nil,
			expectedOutput: "No projects found.",
			expectError:    false,
		},
		{
			name:         "service error",
			mockProjects: nil,
			mockError:    errors.New("database connection failed"),
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock
			mockService := &mocks.MockProjectManager{
				ListFunc: func() ([]*services.Project, error) {
					return tt.mockProjects, tt.mockError
				},
			}
			app.SetProjectServiceForTesting(mockService)

			// Create command and capture output
			cmd := NewCmdProjectList()
			var stdout, stderr bytes.Buffer
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)

			// Execute command
			err := cmd.Execute()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				stdoutStr := stdout.String()

				if tt.expectedOutput != "" {
					assert.Contains(t, stdoutStr, tt.expectedOutput)
				}

				if len(tt.mockProjects) > 0 {
					// Verify project information is included in output
					for _, project := range tt.mockProjects {
						assert.Contains(t, stdoutStr, project.Name)
						assert.Contains(t, stdoutStr, project.GitURL)
					}
				}
			}
		})
	}
}

func TestNewCmdProjectListCommand(t *testing.T) {
	cmd := NewCmdProjectList()

	// Test command configuration
	assert.Equal(t, "list", cmd.Use)
	assert.Equal(t, "List all managed projects", cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotNil(t, cmd.RunE)

	// Verify the command can be found by name
	assert.Equal(t, "list", cmd.Name())
}
