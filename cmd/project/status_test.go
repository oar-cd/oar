package project

import (
	"bytes"
	"errors"
	"testing"

	"github.com/oar-cd/oar/services"

	"github.com/google/uuid"
	"github.com/oar-cd/oar/app"
	"github.com/oar-cd/oar/testing/mocks"
	"github.com/stretchr/testify/assert"
)

func TestNewCmdProjectStatus(t *testing.T) {
	testProjectID := uuid.New()

	tests := []struct {
		name         string
		args         []string
		mockStatus   *services.ComposeStatus
		mockError    error
		expectError  bool
		expectedText string
	}{
		{
			name: "project running status",
			args: []string{testProjectID.String()},
			mockStatus: &services.ComposeStatus{
				Status: services.ComposeProjectStatusRunning,
				Uptime: "2h 30m",
				Containers: []services.ContainerInfo{
					{Service: "web", Status: "Up 2 hours", State: "running"},
					{Service: "db", Status: "Up 2 hours", State: "running"},
				},
			},
			mockError:    nil,
			expectError:  false,
			expectedText: "Status: running",
		},
		{
			name: "project stopped status",
			args: []string{testProjectID.String()},
			mockStatus: &services.ComposeStatus{
				Status:     services.ComposeProjectStatusStopped,
				Containers: []services.ContainerInfo{},
			},
			mockError:    nil,
			expectError:  false,
			expectedText: "Status: stopped",
		},
		{
			name: "project with failed containers",
			args: []string{testProjectID.String()},
			mockStatus: &services.ComposeStatus{
				Status: services.ComposeProjectStatusFailed,
				Containers: []services.ContainerInfo{
					{Service: "web", Status: "Up 1 hour", State: "running"},
					{Service: "db", Status: "Exited (1) 10 minutes ago", State: "exited"},
				},
			},
			mockError:    nil,
			expectError:  false,
			expectedText: "Status: failed",
		},
		{
			name: "project with only successful init containers",
			args: []string{testProjectID.String()},
			mockStatus: &services.ComposeStatus{
				Status: services.ComposeProjectStatusUnknown,
				Containers: []services.ContainerInfo{
					{Service: "migrate", Status: "Exited (0) 5 minutes ago", State: "exited", ExitCode: 0},
					{Service: "collectstatic", Status: "Exited (0) 5 minutes ago", State: "exited", ExitCode: 0},
				},
			},
			mockError:    nil,
			expectError:  false,
			expectedText: "Status: unknown",
		},
		{
			name:        "status error",
			args:        []string{testProjectID.String()},
			mockStatus:  nil,
			mockError:   errors.New("failed to get project status"),
			expectError: true,
		},
		{
			name:        "invalid project ID",
			args:        []string{"invalid-uuid"},
			mockStatus:  nil,
			mockError:   nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock
			mockService := &mocks.MockProjectManager{
				GetStatusFunc: func(projectID uuid.UUID) (*services.ComposeStatus, error) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					return tt.mockStatus, nil
				},
			}
			app.SetProjectServiceForTesting(mockService)

			// Create command and capture output
			cmd := NewCmdProjectStatus()
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

				if tt.mockStatus != nil {
					// Verify uptime is shown for running projects
					if tt.mockStatus.Status == services.ComposeProjectStatusRunning && tt.mockStatus.Uptime != "" {
						assert.Contains(t, stdoutStr, "Uptime:")
						assert.Contains(t, stdoutStr, tt.mockStatus.Uptime)
					}

					// Verify container information is shown
					for _, container := range tt.mockStatus.Containers {
						assert.Contains(t, stdoutStr, container.Service)
						assert.Contains(t, stdoutStr, container.Status)
						if container.State == "running" {
							assert.Contains(t, stdoutStr, "[OK]")
						} else {
							assert.Contains(t, stdoutStr, "[ERROR]")
						}
					}
				}
			}
		})
	}
}

func TestNewCmdProjectStatusCommand(t *testing.T) {
	cmd := NewCmdProjectStatus()

	// Test command configuration
	assert.Equal(t, "status <project-id>", cmd.Use)
	assert.Equal(t, "Show the status of a project's containers", cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotNil(t, cmd.RunE)

	// Verify the command can be found by name
	assert.Equal(t, "status", cmd.Name())
}
