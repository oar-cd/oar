package project

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/oar-cd/oar/services"
	"github.com/oar-cd/oar/testing/mocks"

	"github.com/google/uuid"
	"github.com/oar-cd/oar/app"
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

func TestCmdProjectStopErrorHandling(t *testing.T) {
	testProjectID := uuid.New()
	testProject := &services.Project{
		ID:     testProjectID,
		Name:   "test-project",
		GitURL: "https://github.com/test/project.git",
		Status: services.ProjectStatusRunning,
	}

	tests := []struct {
		name               string
		args               []string
		mockProject        *services.Project
		mockGetError       error
		mockStopError      error
		expectSilenceUsage bool
		expectError        bool
		description        string
	}{
		{
			name:               "no arguments provided - should show usage",
			args:               []string{},
			expectSilenceUsage: false,
			expectError:        true,
			description:        "Argument validation errors should show usage",
		},
		{
			name:               "invalid UUID - runtime validation, should silence usage",
			args:               []string{"invalid-uuid"},
			expectSilenceUsage: true,
			expectError:        true,
			description:        "UUID validation errors happen at runtime, so usage is silenced",
		},
		{
			name:               "project not found - runtime error, should silence usage",
			args:               []string{testProjectID.String()},
			mockGetError:       errors.New("project not found"),
			expectSilenceUsage: true,
			expectError:        true,
			description:        "Runtime errors should not show usage",
		},
		{
			name:               "stop failure - runtime error, should silence usage",
			args:               []string{testProjectID.String()},
			mockProject:        testProject,
			mockStopError:      errors.New("stop failed"),
			expectSilenceUsage: true,
			expectError:        true,
			description:        "Runtime errors should not show usage",
		},
		{
			name:               "successful stop - should not silence usage",
			args:               []string{testProjectID.String()},
			mockProject:        testProject,
			expectSilenceUsage: false,
			expectError:        false,
			description:        "Successful commands should not silence usage",
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

			// Verify error expectation
			if tt.expectError {
				assert.Error(t, err, tt.description)
			} else {
				assert.NoError(t, err, tt.description)
			}

			// Verify SilenceUsage is set correctly
			assert.Equal(t, tt.expectSilenceUsage, cmd.SilenceUsage,
				"SilenceUsage should be %t for case: %s", tt.expectSilenceUsage, tt.description)
		})
	}
}
