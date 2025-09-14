package project

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/oar-cd/oar/services"

	"github.com/google/uuid"
	"github.com/oar-cd/oar/app"
	"github.com/oar-cd/oar/testing/mocks"
	"github.com/stretchr/testify/assert"
)

func TestNewCmdProjectLogs(t *testing.T) {
	testProjectID := uuid.New()
	testProject := &services.Project{
		ID:        testProjectID,
		Name:      "test-project",
		GitURL:    "https://github.com/test/project.git",
		Status:    services.ProjectStatusRunning,
		CreatedAt: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
	}

	tests := []struct {
		name          string
		args          []string
		mockProject   *services.Project
		mockGetError  error
		mockLogsError error
		expectError   bool
		expectedText  string
	}{
		{
			name:          "logs success",
			args:          []string{testProjectID.String()},
			mockProject:   testProject,
			mockGetError:  nil,
			mockLogsError: nil,
			expectError:   false,
			expectedText:  "Streaming logs for project",
		},
		{
			name:         "project not found",
			args:         []string{testProjectID.String()},
			mockProject:  nil,
			mockGetError: errors.New("project not found"),
			expectError:  true,
		},
		{
			name:          "logs error",
			args:          []string{testProjectID.String()},
			mockProject:   testProject,
			mockGetError:  nil,
			mockLogsError: errors.New("failed to get project logs"),
			expectError:   true,
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
			mockService := &mocks.MockProjectManager{
				GetFunc: func(id uuid.UUID) (*services.Project, error) {
					if tt.mockGetError != nil {
						return nil, tt.mockGetError
					}
					return tt.mockProject, nil
				},
				GetLogsPipingFunc: func(projectID uuid.UUID) error {
					return tt.mockLogsError
				},
			}
			app.SetProjectServiceForTesting(mockService)

			// Create command and capture output
			cmd := NewCmdProjectLogs()
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

				if tt.mockProject != nil && tt.mockLogsError == nil {
					// Verify logs info is shown
					assert.Contains(t, stdoutStr, tt.mockProject.Name)
					assert.Contains(t, stdoutStr, "Press Ctrl+C to stop")
				}
			}
		})
	}
}

func TestNewCmdProjectLogsCommand(t *testing.T) {
	cmd := NewCmdProjectLogs()

	// Test command configuration
	assert.Equal(t, "logs <project-id>", cmd.Use)
	assert.Equal(t, "View logs from a project's containers", cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotNil(t, cmd.RunE)

	// Verify the command can be found by name
	assert.Equal(t, "logs", cmd.Name())
}
