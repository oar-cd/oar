package project

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/oar-cd/oar/internal/app"
	"github.com/oar-cd/oar/services"
	"github.com/oar-cd/oar/testing/mocks"
	"github.com/stretchr/testify/assert"
)

func TestNewCmdProjectDeployments(t *testing.T) {
	testProjectID := uuid.New()
	testProject := &services.Project{
		ID:         testProjectID,
		Name:       "test-project",
		GitURL:     "https://github.com/test/project.git",
		WorkingDir: "/tmp/test-working-dir",
		CreatedAt:  time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
	}

	testDeployments := []*services.Deployment{
		{
			ID:         uuid.New(),
			ProjectID:  testProjectID,
			CommitHash: "abc123def456",
			Status:     services.DeploymentStatusCompleted,
			CreatedAt:  time.Date(2023, 1, 1, 14, 0, 0, 0, time.UTC),
			UpdatedAt:  time.Date(2023, 1, 1, 14, 5, 0, 0, time.UTC),
		},
		{
			ID:         uuid.New(),
			ProjectID:  testProjectID,
			CommitHash: "def456ghi789",
			Status:     services.DeploymentStatusFailed,
			CreatedAt:  time.Date(2023, 1, 1, 16, 0, 0, 0, time.UTC),
			UpdatedAt:  time.Date(2023, 1, 1, 16, 2, 0, 0, time.UTC),
		},
	}

	tests := []struct {
		name            string
		args            []string
		mockProject     *services.Project
		mockDeployments []*services.Deployment
		mockError       error
		expectedOutput  string
		expectError     bool
	}{
		{
			name:            "deployments success",
			args:            []string{testProjectID.String()},
			mockProject:     testProject,
			mockDeployments: testDeployments,
			expectedOutput:  "ID", // Should contain table headers
			expectError:     false,
		},
		{
			name:            "no deployments found",
			args:            []string{testProjectID.String()},
			mockProject:     testProject,
			mockDeployments: []*services.Deployment{},
			expectedOutput:  "No deployments found for project 'test-project'",
			expectError:     false,
		},
		{
			name:        "project not found",
			args:        []string{testProjectID.String()},
			mockError:   errors.New("project not found"),
			expectError: true,
		},
		{
			name:        "deployments error",
			args:        []string{testProjectID.String()},
			mockProject: testProject,
			mockError:   errors.New("database error"),
			expectError: true,
		},
		{
			name:        "invalid project ID",
			args:        []string{"invalid-uuid"},
			expectError: true,
		},
		{
			name:        "no project ID provided",
			args:        []string{},
			expectError: false, // Should show help instead of error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock project manager
			mockProjectManager := &mocks.MockProjectManager{}

			// Set up expectations
			if len(tt.args) > 0 {
				projectID, err := uuid.Parse(tt.args[0])
				if err == nil {
					if tt.mockError != nil {
						mockProjectManager.GetFunc = func(id uuid.UUID) (*services.Project, error) {
							if id == projectID {
								return nil, tt.mockError
							}
							return nil, errors.New("unexpected project ID")
						}
						if tt.mockProject != nil {
							mockProjectManager.ListDeploymentsFunc = func(id uuid.UUID) ([]*services.Deployment, error) {
								if id == projectID {
									return nil, tt.mockError
								}
								return nil, errors.New("unexpected project ID")
							}
						}
					} else {
						mockProjectManager.GetFunc = func(id uuid.UUID) (*services.Project, error) {
							if id == projectID {
								return tt.mockProject, nil
							}
							return nil, errors.New("project not found")
						}
						mockProjectManager.ListDeploymentsFunc = func(id uuid.UUID) ([]*services.Deployment, error) {
							if id == projectID {
								return tt.mockDeployments, nil
							}
							return nil, errors.New("project not found")
						}
					}
				}
			}

			// Replace the project service temporarily
			app.SetProjectServiceForTesting(mockProjectManager)

			// Create command and set up output buffer
			cmd := NewCmdProjectDeployments()
			buf := &bytes.Buffer{}
			cmd.SetOut(buf)
			cmd.SetErr(buf)

			// Set arguments
			cmd.SetArgs(tt.args)

			// Execute command
			err := cmd.Execute()

			// Check results
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				output := buf.String()
				assert.Contains(t, output, tt.expectedOutput)
			}
		})
	}
}

func TestNewCmdProjectDeploymentsCommand(t *testing.T) {
	cmd := NewCmdProjectDeployments()

	assert.NotNil(t, cmd)
	assert.Equal(t, "deployments <project-id>", cmd.Use)
	assert.Equal(t, "List deployments for a project", cmd.Short)
	assert.Contains(t, cmd.Long, "Display all deployments for a specific project")
}
