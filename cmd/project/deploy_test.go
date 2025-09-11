package project

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/oar-cd/oar/services"
	"github.com/oar-cd/oar/testing/mocks"

	"github.com/google/uuid"
	"github.com/oar-cd/oar/internal/app"
	"github.com/stretchr/testify/assert"
)

func TestNewCmdProjectDeploy(t *testing.T) {
	testProjectID := uuid.New()
	lastCommit := "abc123def456"
	testProject := &services.Project{
		ID:         testProjectID,
		Name:       "test-project",
		GitURL:     "https://github.com/test/project.git",
		Status:     services.ProjectStatusStopped,
		LastCommit: &lastCommit,
		CreatedAt:  time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
	}
	deployedProject := &services.Project{
		ID:         testProjectID,
		Name:       "test-project",
		GitURL:     "https://github.com/test/project.git",
		Status:     services.ProjectStatusRunning,
		LastCommit: &lastCommit,
		CreatedAt:  time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
	}

	tests := []struct {
		name               string
		args               []string
		flags              map[string]interface{}
		mockProject        *services.Project
		mockUpdatedProject *services.Project
		mockGetError       error
		mockDeployError    error
		expectError        bool
		expectedText       string
	}{
		{
			name:               "deploy success with pull",
			args:               []string{testProjectID.String()},
			flags:              nil, // pull defaults to true
			mockProject:        testProject,
			mockUpdatedProject: deployedProject,
			mockGetError:       nil,
			mockDeployError:    nil,
			expectError:        false,
			expectedText:       "deployed successfully",
		},
		{
			name: "deploy success without pull",
			args: []string{testProjectID.String()},
			flags: map[string]interface{}{
				"pull": false,
			},
			mockProject:        testProject,
			mockUpdatedProject: deployedProject,
			mockGetError:       nil,
			mockDeployError:    nil,
			expectError:        false,
			expectedText:       "Git pull: disabled",
		},
		{
			name:            "project not found",
			args:            []string{testProjectID.String()},
			mockProject:     nil,
			mockGetError:    errors.New("project not found"),
			mockDeployError: nil,
			expectError:     true,
		},
		{
			name:            "deploy error",
			args:            []string{testProjectID.String()},
			mockProject:     testProject,
			mockGetError:    nil,
			mockDeployError: errors.New("failed to deploy project"),
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
					// Second call after deployment should return updated project
					if tt.mockUpdatedProject != nil {
						return tt.mockUpdatedProject, nil
					}
					return tt.mockProject, nil
				},
				DeployPipingFunc: func(projectID uuid.UUID, pull bool) error {
					return tt.mockDeployError
				},
			}
			app.SetProjectServiceForTesting(mockService)

			// Create command and capture output
			cmd := NewCmdProjectDeploy()
			var stdout, stderr bytes.Buffer
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)
			cmd.SetArgs(tt.args)

			// Set flags if provided
			if tt.flags != nil {
				for flag, value := range tt.flags {
					switch v := value.(type) {
					case bool:
						if v {
							_ = cmd.Flags().Set(flag, "true")
						} else {
							_ = cmd.Flags().Set(flag, "false")
						}
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
				stdoutStr := stdout.String()

				if tt.expectedText != "" {
					assert.Contains(t, stdoutStr, tt.expectedText)
				}

				if tt.mockProject != nil && tt.mockDeployError == nil {
					// Verify deployment info is shown
					assert.Contains(t, stdoutStr, tt.mockProject.Name)
					assert.Contains(t, stdoutStr, tt.mockProject.GitURL)
				}
			}
		})
	}
}

func TestNewCmdProjectDeployCommand(t *testing.T) {
	cmd := NewCmdProjectDeploy()

	// Test command configuration
	assert.Equal(t, "deploy <project-id>", cmd.Use)
	assert.Equal(t, "Deploy or update a project", cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotNil(t, cmd.RunE)

	// Test flag exists
	pullFlag := cmd.Flags().Lookup("pull")
	assert.NotNil(t, pullFlag)
	assert.Equal(t, "true", pullFlag.DefValue) // Default should be true

	// Verify the command can be found by name
	assert.Equal(t, "deploy", cmd.Name())
}
