package output

import (
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/fatih/color"
	"github.com/google/uuid"
	"github.com/oar-cd/oar/services"
	"github.com/stretchr/testify/assert"
)

func TestInitColors(t *testing.T) {
	tests := []struct {
		name       string
		setNoColor bool
	}{
		{
			name:       "colors enabled",
			setNoColor: false,
		},
		{
			name:       "colors disabled by NO_COLOR",
			setNoColor: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original state
			originalNoColor := color.NoColor
			originalMaybeColorize := maybeColorize

			// Set up test state
			color.NoColor = tt.setNoColor
			maybeColorize = nil

			// Test InitColors
			InitColors()

			// Verify maybeColorize was set
			assert.NotNil(t, maybeColorize)

			// Test the function works
			result := maybeColorize(Success, "test %s", "message")
			assert.Contains(t, result, "test message")

			// Restore original state
			color.NoColor = originalNoColor
			maybeColorize = originalMaybeColorize
		})
	}
}

func TestPrintMessage(t *testing.T) {
	// Save original state
	originalMaybeColorize := maybeColorize

	tests := []struct {
		name         string
		kind         color.Attribute
		template     string
		args         []any
		setColorizer bool
		expected     string
	}{
		{
			name:         "plain message with nil colorizer",
			kind:         Plain,
			template:     "Hello %s",
			args:         []any{"World"},
			setColorizer: false,
			expected:     "Hello World\n",
		},
		{
			name:         "plain message with colorizer",
			kind:         Plain,
			template:     "Plain %s",
			args:         []any{"text"},
			setColorizer: true,
			expected:     "Plain text\n",
		},
		{
			name:         "success message with colorizer",
			kind:         Success,
			template:     "Success %s",
			args:         []any{"message"},
			setColorizer: true,
			expected:     "Success message", // Will have colors but we just check content
		},
		{
			name:         "no args",
			kind:         Plain,
			template:     "Simple message",
			args:         nil,
			setColorizer: false,
			expected:     "Simple message\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setColorizer {
				maybeColorize = func(kind color.Attribute, tmpl string, a ...any) string {
					return color.New(kind).SprintfFunc()(tmpl, a...)
				}
			} else {
				maybeColorize = nil
			}

			result := PrintMessage(tt.kind, tt.template, tt.args...)
			if tt.kind == Plain || !tt.setColorizer {
				assert.Equal(t, tt.expected, result)
			} else {
				// For colored output, just check it contains the expected text
				assert.Contains(t, result, "Success message")
			}
		})
	}

	// Restore original state
	maybeColorize = originalMaybeColorize
}

func TestPrintTable(t *testing.T) {
	tests := []struct {
		name        string
		header      []string
		data        [][]string
		expectError bool
	}{
		{
			name:   "simple table with header",
			header: []string{"Column1", "Column2"},
			data: [][]string{
				{"Row1Col1", "Row1Col2"},
				{"Row2Col1", "Row2Col2"},
			},
			expectError: false,
		},
		{
			name:   "table without header",
			header: []string{},
			data: [][]string{
				{"Value1", "Value2"},
			},
			expectError: false,
		},
		{
			name:        "empty table",
			header:      []string{"Header1", "Header2"},
			data:        [][]string{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := PrintTable(tt.header, tt.data)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, result)
				// Check that data appears in result
				for _, row := range tt.data {
					for _, cell := range row {
						if cell != "" {
							assert.Contains(t, result, cell)
						}
					}
				}
			}
		})
	}
}

func TestPrintProjectDetails(t *testing.T) {
	projectID := uuid.New()
	createdAt := time.Date(2023, 1, 15, 10, 30, 0, 0, time.UTC)
	updatedAt := time.Date(2023, 1, 16, 14, 45, 0, 0, time.UTC)

	tests := []struct {
		name     string
		project  *services.Project
		short    bool
		expected []string
	}{
		{
			name: "detailed project with HTTP auth",
			project: &services.Project{
				ID:           projectID,
				Name:         "test-project",
				Status:       services.ProjectStatusRunning,
				GitURL:       "https://github.com/test/repo",
				WorkingDir:   "/tmp/projects/test-project",
				ComposeFiles: []string{"docker-compose.yml", "docker-compose.prod.yml"},
				Variables:    []string{"ENV=production", "PORT=8080"},
				GitAuth: &services.GitAuthConfig{
					HTTPAuth: &services.GitHTTPAuthConfig{
						Username: "token",
						Password: "github_pat_123456789",
					},
				},
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
			},
			short: false,
			expected: []string{
				"test-project",
				"running",
				"https://github.com/test/repo",
				"HTTP",
				"token",
				"docker-compose.yml",
				"docker-compose.prod.yml",
			},
		},
		{
			name: "short project details",
			project: &services.Project{
				ID:         projectID,
				Name:       "short-project",
				Status:     services.ProjectStatusStopped,
				GitURL:     "https://github.com/test/short",
				WorkingDir: "/tmp/projects/short-project",
			},
			short:    true,
			expected: []string{"short-project", "stopped", "https://github.com/test/short"},
		},
		{
			name: "project with SSH auth",
			project: &services.Project{
				ID:         projectID,
				Name:       "ssh-project",
				Status:     services.ProjectStatusError,
				GitURL:     "git@github.com:test/repo.git",
				WorkingDir: "/tmp/projects/ssh-project",
				GitAuth: &services.GitAuthConfig{
					SSHAuth: &services.GitSSHAuthConfig{
						User:       "git",
						PrivateKey: "-----BEGIN OPENSSH PRIVATE KEY-----\ntest key\n-----END OPENSSH PRIVATE KEY-----",
					},
				},
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
			},
			short:    false,
			expected: []string{"ssh-project", "error", "git@github.com:test/repo.git", "SSH", "git"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := PrintProjectDetails(tt.project, tt.short)
			assert.NoError(t, err)
			assert.NotEmpty(t, result)

			// Check that expected strings appear in result
			for _, expected := range tt.expected {
				assert.Contains(t, result, expected, "Expected %q to be in result", expected)
			}
		})
	}
}

func TestGetAuthenticationInfo(t *testing.T) {
	tests := []struct {
		name           string
		project        *services.Project
		expectedMethod string
		expectedUser   string
	}{
		{
			name: "no authentication",
			project: &services.Project{
				GitAuth: nil,
			},
			expectedMethod: "None",
			expectedUser:   "",
		},
		{
			name: "HTTP authentication",
			project: &services.Project{
				GitAuth: &services.GitAuthConfig{
					HTTPAuth: &services.GitHTTPAuthConfig{
						Username: "testuser",
						Password: "testpass",
					},
				},
			},
			expectedMethod: "HTTP",
			expectedUser:   "testuser",
		},
		{
			name: "SSH authentication",
			project: &services.Project{
				GitAuth: &services.GitAuthConfig{
					SSHAuth: &services.GitSSHAuthConfig{
						User:       "git",
						PrivateKey: "ssh-key",
					},
				},
			},
			expectedMethod: "SSH",
			expectedUser:   "git",
		},
		{
			name: "empty auth config",
			project: &services.Project{
				GitAuth: &services.GitAuthConfig{},
			},
			expectedMethod: "None",
			expectedUser:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			method, user := getAuthenticationInfo(tt.project)
			assert.Equal(t, tt.expectedMethod, method)
			assert.Equal(t, tt.expectedUser, user)
		})
	}
}

func TestGetAuthenticationCredential(t *testing.T) {
	tests := []struct {
		name     string
		project  *services.Project
		expected string
	}{
		{
			name: "no authentication",
			project: &services.Project{
				GitAuth: nil,
			},
			expected: "",
		},
		{
			name: "HTTP auth with password",
			project: &services.Project{
				GitAuth: &services.GitAuthConfig{
					HTTPAuth: &services.GitHTTPAuthConfig{
						Username: "user",
						Password: "secret123456",
					},
				},
			},
			expected: "sec******456",
		},
		{
			name: "HTTP auth with empty password",
			project: &services.Project{
				GitAuth: &services.GitAuthConfig{
					HTTPAuth: &services.GitHTTPAuthConfig{
						Username: "user",
						Password: "",
					},
				},
			},
			expected: "(not set)",
		},
		{
			name: "SSH auth with key",
			project: &services.Project{
				GitAuth: &services.GitAuthConfig{
					SSHAuth: &services.GitSSHAuthConfig{
						User:       "git",
						PrivateKey: "some-ssh-key",
					},
				},
			},
			expected: "SSH Private Key (***masked***)",
		},
		{
			name: "SSH auth with empty key",
			project: &services.Project{
				GitAuth: &services.GitAuthConfig{
					SSHAuth: &services.GitSSHAuthConfig{
						User:       "git",
						PrivateKey: "",
					},
				},
			},
			expected: "(not set)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getAuthenticationCredential(tt.project)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPrintProjectList(t *testing.T) {
	projectID1 := uuid.New()
	projectID2 := uuid.New()
	createdAt := time.Date(2023, 1, 15, 10, 30, 0, 0, time.UTC)
	updatedAt := time.Date(2023, 1, 16, 14, 45, 0, 0, time.UTC)

	// Set up colors for testing
	InitColors()

	tests := []struct {
		name     string
		projects []*services.Project
		expected []string
	}{
		{
			name:     "empty project list",
			projects: []*services.Project{},
			expected: []string{"No projects found"},
		},
		{
			name: "single project",
			projects: []*services.Project{
				{
					ID:        projectID1,
					Name:      "test-project",
					Status:    services.ProjectStatusRunning,
					GitURL:    "https://github.com/test/repo",
					CreatedAt: createdAt,
					UpdatedAt: updatedAt,
				},
			},
			expected: []string{"test-project", "running", "https://github.com/test/repo"},
		},
		{
			name: "multiple projects",
			projects: []*services.Project{
				{
					ID:        projectID1,
					Name:      "project-1",
					Status:    services.ProjectStatusRunning,
					GitURL:    "https://github.com/test/project1",
					CreatedAt: createdAt,
					UpdatedAt: updatedAt,
				},
				{
					ID:        projectID2,
					Name:      "project-2",
					Status:    services.ProjectStatusStopped,
					GitURL:    "https://github.com/test/project2",
					CreatedAt: createdAt,
					UpdatedAt: updatedAt,
				},
			},
			expected: []string{"project-1", "running", "project-2", "stopped"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := PrintProjectList(tt.projects)
			assert.NoError(t, err)
			assert.NotEmpty(t, result)

			// Check that expected strings appear in result
			for _, expected := range tt.expected {
				assert.Contains(t, result, expected, "Expected %q to be in result", expected)
			}
		})
	}
}

func TestFormatProjectStatus(t *testing.T) {
	// Set up colors for testing
	InitColors()

	tests := []struct {
		name     string
		status   string
		contains string // We check contains rather than exact match due to color codes
	}{
		{
			name:     "running status",
			status:   "running",
			contains: "running",
		},
		{
			name:     "Running status (capitalized)",
			status:   "Running",
			contains: "Running",
		},
		{
			name:     "stopped status",
			status:   "stopped",
			contains: "stopped",
		},
		{
			name:     "error status",
			status:   "error",
			contains: "error",
		},
		{
			name:     "unknown status",
			status:   "unknown",
			contains: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatProjectStatus(tt.status)
			assert.Contains(t, result, tt.contains)
		})
	}

	// Test with colors disabled
	maybeColorize = nil
	result := formatProjectStatus("running")
	assert.Equal(t, "running", result)
}

func TestFprintFunctions(t *testing.T) {
	// Set up colors for testing
	InitColors()

	buf := &bytes.Buffer{}

	// Test Fprint
	err := Fprint(buf, Success, "Test %s", "message")
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Test message")

	// Test with mock command
	mockCmd := &mockCommand{buf: &bytes.Buffer{}}

	tests := []struct {
		name     string
		fn       func() error
		expected string
	}{
		{
			name:     "FprintPlain",
			fn:       func() error { return FprintPlain(mockCmd, "Plain %s", "text") },
			expected: "Plain text",
		},
		{
			name:     "FprintSuccess",
			fn:       func() error { return FprintSuccess(mockCmd, "Success %s", "text") },
			expected: "Success text",
		},
		{
			name:     "FprintWarning",
			fn:       func() error { return FprintWarning(mockCmd, "Warning %s", "text") },
			expected: "Warning text",
		},
		{
			name:     "FprintError",
			fn:       func() error { return FprintError(mockCmd, "Error %s", "text") },
			expected: "Error text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCmd.buf.Reset()
			err := tt.fn()
			assert.NoError(t, err)
			assert.Contains(t, mockCmd.buf.String(), tt.expected)
		})
	}
}

func TestPrintDeploymentList(t *testing.T) {
	deploymentID1 := uuid.New()
	deploymentID2 := uuid.New()
	deploymentID3 := uuid.New()
	createdAt := time.Date(2023, 1, 15, 10, 30, 0, 0, time.UTC)
	updatedAt := time.Date(2023, 1, 16, 14, 45, 0, 0, time.UTC)

	// Set up colors for testing
	InitColors()

	tests := []struct {
		name        string
		deployments []*services.Deployment
		projectName string
		expected    []string
	}{
		{
			name:        "empty deployment list",
			deployments: []*services.Deployment{},
			projectName: "test-project",
			expected:    []string{"No deployments found for project 'test-project'."},
		},
		{
			name: "single deployment",
			deployments: []*services.Deployment{
				{
					ID:         deploymentID1,
					Status:     services.DeploymentStatusCompleted,
					CommitHash: "abc123def456",
					CreatedAt:  createdAt,
					UpdatedAt:  updatedAt,
				},
			},
			projectName: "test-project",
			expected: []string{
				"ID",
				"STATUS",
				"COMMIT",
				"CREATED AT",
				"UPDATED AT",
				deploymentID1.String(),
				"completed",
				"abc123de",
				"2023-01-15 10:30:00",
				"2023-01-16 14:45:00",
			},
		},
		{
			name: "multiple deployments with different statuses",
			deployments: []*services.Deployment{
				{
					ID:         deploymentID1,
					Status:     services.DeploymentStatusCompleted,
					CommitHash: "abc123def456",
					CreatedAt:  createdAt,
					UpdatedAt:  updatedAt,
				},
				{
					ID:         deploymentID2,
					Status:     services.DeploymentStatusStarted,
					CommitHash: "def456ghi789",
					CreatedAt:  createdAt,
					UpdatedAt:  updatedAt,
				},
				{
					ID:         deploymentID3,
					Status:     services.DeploymentStatusFailed,
					CommitHash: "ghi789jkl012",
					CreatedAt:  createdAt,
					UpdatedAt:  updatedAt,
				},
			},
			projectName: "test-project",
			expected: []string{
				"ID",
				"STATUS",
				"COMMIT",
				"CREATED AT",
				"UPDATED AT",
				deploymentID1.String(),
				"completed",
				"abc123de",
				"2023-01-15 10:30:00",
				"2023-01-16 14:45:00",
				deploymentID2.String(),
				"started",
				"def456gh",
				"2023-01-15 10:30:00",
				"2023-01-16 14:45:00",
				deploymentID3.String(),
				"failed",
				"ghi789jk",
				"2023-01-15 10:30:00",
				"2023-01-16 14:45:00",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := PrintDeploymentList(test.deployments, test.projectName)

			assert.NoError(t, err)
			for _, expectedStr := range test.expected {
				assert.Contains(t, result, expectedStr)
			}
		})
	}
}

func TestFormatDeploymentStatus(t *testing.T) {
	// Set up colors for testing
	InitColors()

	tests := []struct {
		name     string
		status   string
		expected string
	}{
		{
			name:     "completed status",
			status:   "completed",
			expected: "completed",
		},
		{
			name:     "started status",
			status:   "started",
			expected: "started",
		},
		{
			name:     "failed status",
			status:   "failed",
			expected: "failed",
		},
		{
			name:     "unknown status",
			status:   "unknown",
			expected: "unknown",
		},
		{
			name:     "case insensitive - COMPLETED",
			status:   "COMPLETED",
			expected: "COMPLETED",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := formatDeploymentStatus(test.status)

			// Since colors are disabled, result should be the same as input
			assert.Equal(t, test.expected, result)
		})
	}
}

// mockCommand implements the interface needed for FprintCmd functions
type mockCommand struct {
	buf *bytes.Buffer
}

func (m *mockCommand) OutOrStdout() io.Writer {
	return m.buf
}
