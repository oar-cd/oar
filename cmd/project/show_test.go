package project

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/oar-cd/oar/services"

	"github.com/google/uuid"
	"github.com/oar-cd/oar/internal/app"
	"github.com/oar-cd/oar/testing/mocks"
	"github.com/stretchr/testify/assert"
)

func TestNewCmdProjectShow(t *testing.T) {
	testProjectID := uuid.New()
	testProject := &services.Project{
		ID:         testProjectID,
		Name:       "test-project",
		GitURL:     "https://github.com/test/project.git",
		WorkingDir: "/tmp/test-working-dir",
		CreatedAt:  time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
	}

	tests := []struct {
		name        string
		args        []string
		mockProject *services.Project
		mockError   error
		expectError bool
	}{
		{
			name:        "show project success",
			args:        []string{testProjectID.String()},
			mockProject: testProject,
			mockError:   nil,
			expectError: false,
		},
		{
			name:        "project not found",
			args:        []string{testProjectID.String()},
			mockProject: nil,
			mockError:   errors.New("project not found"),
			expectError: true,
		},
		{
			name:        "invalid project ID",
			args:        []string{"invalid-uuid"},
			mockProject: nil,
			mockError:   nil,
			expectError: true,
		},
		{
			name:        "no project ID provided",
			args:        []string{},
			mockProject: nil,
			mockError:   nil,
			expectError: false, // Should show help instead of error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock
			mockService := &mocks.MockProjectManager{
				GetFunc: func(id uuid.UUID) (*services.Project, error) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					return tt.mockProject, nil
				},
			}
			app.SetProjectServiceForTesting(mockService)

			// Create command and capture output
			cmd := NewCmdProjectShow()
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

				if tt.mockProject != nil {
					stdoutStr := stdout.String()
					// Verify project information is included in output
					assert.Contains(t, stdoutStr, tt.mockProject.Name)
					assert.Contains(t, stdoutStr, tt.mockProject.GitURL)
				}
			}
		})
	}
}

func TestNewCmdProjectShowCommand(t *testing.T) {
	cmd := NewCmdProjectShow()

	// Test command configuration
	assert.Equal(t, "show <project-id>", cmd.Use)
	assert.Equal(t, "Show detailed project information", cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotNil(t, cmd.RunE)

	// Verify the command can be found by name
	assert.Equal(t, "show", cmd.Name())
}
