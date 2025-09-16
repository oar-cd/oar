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

func TestNewCmdProjectRemove(t *testing.T) {
	testProjectID := uuid.New()
	testProject := &services.Project{
		ID:         testProjectID,
		Name:       "test-project",
		GitURL:     "https://github.com/test/project.git",
		Status:     services.ProjectStatusStopped,
		WorkingDir: "/tmp/test-project",
		CreatedAt:  time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
	}

	tests := []struct {
		name            string
		args            []string
		flags           map[string]interface{}
		mockProject     *services.Project
		mockGetError    error
		mockRemoveError error
		expectError     bool
	}{
		{
			name: "remove project success with confirm flag",
			args: []string{testProjectID.String()},
			flags: map[string]interface{}{
				"confirm": true,
			},
			mockProject:     testProject,
			mockGetError:    nil,
			mockRemoveError: nil,
			expectError:     false,
		},
		{
			name:         "project not found",
			args:         []string{testProjectID.String()},
			mockProject:  nil,
			mockGetError: errors.New("project not found"),
			expectError:  true,
		},
		{
			name: "remove service error",
			args: []string{testProjectID.String()},
			flags: map[string]interface{}{
				"confirm": true,
			},
			mockProject:     testProject,
			mockGetError:    nil,
			mockRemoveError: errors.New("failed to remove project"),
			expectError:     true,
		},
		{
			name: "running project without force flag",
			args: []string{testProjectID.String()},
			flags: map[string]interface{}{
				"confirm": true,
			},
			mockProject: &services.Project{
				ID:         testProjectID,
				Name:       "running-project",
				GitURL:     "https://github.com/test/project.git",
				Status:     services.ProjectStatusRunning,
				WorkingDir: "/tmp/test-project",
				CreatedAt:  time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
			},
			mockGetError:    nil,
			mockRemoveError: nil,
			expectError:     true,
		},
		{
			name: "running project with force flag",
			args: []string{testProjectID.String()},
			flags: map[string]interface{}{
				"confirm": true,
				"force":   true,
			},
			mockProject: &services.Project{
				ID:         testProjectID,
				Name:       "running-project",
				GitURL:     "https://github.com/test/project.git",
				Status:     services.ProjectStatusRunning,
				WorkingDir: "/tmp/test-project",
				CreatedAt:  time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
			},
			mockGetError:    nil,
			mockRemoveError: nil,
			expectError:     false,
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
				RemoveFunc: func(projectID uuid.UUID, removeVolumes bool) error {
					return tt.mockRemoveError
				},
			}
			app.SetProjectServiceForTesting(mockService)

			// Create command and capture output
			cmd := NewCmdProjectRemove()
			var stdout, stderr bytes.Buffer
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)
			cmd.SetArgs(tt.args)

			// Set flags if provided
			if tt.flags != nil {
				for flag, value := range tt.flags {
					switch v := value.(type) {
					case bool:
						_ = cmd.Flags().Set(flag, "true")
					case string:
						_ = cmd.Flags().Set(flag, v)
					}
				}
			}

			// Execute command
			err := cmd.Execute()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				if tt.mockProject != nil && tt.mockRemoveError == nil {
					stdoutStr := stdout.String()
					// Verify success message is included
					assert.Contains(t, stdoutStr, "removed successfully")
				}
			}
		})
	}
}

func TestNewCmdProjectRemoveCommand(t *testing.T) {
	cmd := NewCmdProjectRemove()

	// Test command configuration
	assert.Equal(t, "remove <project-id>", cmd.Use)
	assert.Equal(t, "Remove a project and its data", cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotNil(t, cmd.RunE)

	// Test flags exist
	assert.NotNil(t, cmd.Flags().Lookup("confirm"))
	assert.NotNil(t, cmd.Flags().Lookup("force"))

	// Verify the command can be found by name
	assert.Equal(t, "remove", cmd.Name())
}
