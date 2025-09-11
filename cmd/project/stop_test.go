package project

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/ch00k/oar/services"
	"github.com/ch00k/oar/testing/mocks"

	"github.com/ch00k/oar/internal/app"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewCmdProjectStop(t *testing.T) {
	testProjectID := uuid.New()
	runningProject := &services.Project{
		ID:        testProjectID,
		Name:      "test-project",
		GitURL:    "https://github.com/test/project.git",
		Status:    services.ProjectStatusRunning,
		CreatedAt: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
	}
	stoppedProject := &services.Project{
		ID:        testProjectID,
		Name:      "test-project",
		GitURL:    "https://github.com/test/project.git",
		Status:    services.ProjectStatusStopped,
		CreatedAt: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
	}

	tests := []struct {
		name               string
		args               []string
		mockProject        *services.Project
		mockUpdatedProject *services.Project
		mockGetError       error
		mockStopError      error
		expectError        bool
		expectedText       string
	}{
		{
			name:               "stop success",
			args:               []string{testProjectID.String()},
			mockProject:        runningProject,
			mockUpdatedProject: stoppedProject,
			mockGetError:       nil,
			mockStopError:      nil,
			expectError:        false,
			expectedText:       "stopped successfully",
		},
		{
			name:         "project not found",
			args:         []string{testProjectID.String()},
			mockProject:  nil,
			mockGetError: errors.New("project not found"),
			expectError:  true,
		},
		{
			name:          "stop error",
			args:          []string{testProjectID.String()},
			mockProject:   runningProject,
			mockGetError:  nil,
			mockStopError: errors.New("failed to stop project"),
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
			getCalls := 0
			mockService := &mocks.MockProjectManager{
				GetFunc: func(id uuid.UUID) (*services.Project, error) {
					if tt.mockGetError != nil {
						return nil, tt.mockGetError
					}
					getCalls++
					if getCalls == 1 {
						return tt.mockProject, nil
					}
					// Second call after stop should return updated project
					if tt.mockUpdatedProject != nil {
						return tt.mockUpdatedProject, nil
					}
					return tt.mockProject, nil
				},
				StopPipingFunc: func(projectID uuid.UUID) error {
					return tt.mockStopError
				},
			}
			app.SetProjectServiceForTesting(mockService)

			// Create command and capture output
			cmd := NewCmdProjectStop()
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

				if tt.mockProject != nil && tt.mockStopError == nil {
					// Verify stop info is shown
					assert.Contains(t, stdoutStr, tt.mockProject.Name)
				}
			}
		})
	}
}

func TestNewCmdProjectStopCommand(t *testing.T) {
	cmd := NewCmdProjectStop()

	// Test command configuration
	assert.Equal(t, "stop <project-id>", cmd.Use)
	assert.Equal(t, "Stop a running project", cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotNil(t, cmd.RunE)

	// Verify the command can be found by name
	assert.Equal(t, "stop", cmd.Name())
}
