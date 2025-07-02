package project

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ch00k/oar/internal/app"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testOptions struct {
	gitURL      *string  // nil means don't include flag
	name        *string  // nil means don't include flag
	composeFile []string // empty means don't include flag
	envFile     []string // empty means don't include flag
}

func TestProjectAdd(t *testing.T) {
	tests := []struct {
		name        string
		setupRepo   bool
		options     testOptions
		expectError bool
	}{
		{
			name:      "Success - Valid Git Repository",
			setupRepo: true,
			options: testOptions{
				gitURL: stringPtr(""), // empty string means use test repo
				name:   stringPtr("test-project"),
			},
			expectError: false,
		},
		{
			name:      "Error - Missing Git URL",
			setupRepo: false,
			options: testOptions{
				name: stringPtr("test-project"),
				// gitURL is nil, so flag won't be included
			},
			expectError: true,
		},
		//{
		//    name:      "Error - Missing Name",
		//    setupRepo: true,
		//    options: testOptions{
		//        gitURL: stringPtr(""),
		//        // name is nil, so flag won't be included
		//    },
		//    expectError: true,
		//},
		{
			name:      "Success - With Custom Compose Files",
			setupRepo: true,
			options: testOptions{
				gitURL:      stringPtr(""),
				name:        stringPtr("custom-project"),
				composeFile: []string{"docker-compose.yml", "docker-compose.override.yml"},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for test data
			tempDir, err := os.MkdirTemp("", "oar-test-*")
			require.NoError(t, err)
			defer os.RemoveAll(tempDir) // nolint: errcheck

			var repoDir string
			if tt.setupRepo {
				repoDir = setupTestGitRepo(t, tempDir)
			}

			// Initialize the app with test data directory
			err = app.Initialize(tempDir)
			require.NoError(t, err)

			// Build args from options (don't include "project add" prefix)
			args := buildArgsFromOptionsForSingleCommand(tt.options, repoDir)

			var stdout, stderr bytes.Buffer
			rootCmd := NewCmdProjectAdd()
			rootCmd.SetOut(&stdout)
			rootCmd.SetErr(&stderr)
			rootCmd.SetArgs(args)

			// Debug: Print the command being executed
			t.Logf("Executing command: oar %s", strings.Join(args, " "))
			// t.Logf("Executing command: oar %s", strings.Join(rootCmd.Flags().Args(), " "))

			err = rootCmd.Execute()

			// Verify results
			if tt.expectError {
				assert.Error(t, err, "Expected command to fail")
			} else {
				assert.NoError(t, err, "Expected command to succeed")
				// Note: CLI output may not be captured in test buffers since it goes directly to os.Stdout
				// For now, we're just verifying the command succeeds without error
			}
		})
	}
}

// buildArgsFromOptionsForSingleCommand converts testOptions to command line arguments without command prefix
func buildArgsFromOptionsForSingleCommand(opts testOptions, repoDir string) []string {
	var args []string

	if opts.gitURL != nil {
		gitURL := *opts.gitURL
		if gitURL == "" {
			gitURL = repoDir // use test repo path
		}
		args = append(args, "--git-url", gitURL)
	}

	if opts.name != nil {
		args = append(args, "--name", *opts.name)
	}

	for _, file := range opts.composeFile {
		args = append(args, "--compose-file", file)
	}

	for _, file := range opts.envFile {
		args = append(args, "--env-file", file)
	}

	return args
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}

// setupTestGitRepo creates a test git repository with a compose file
func setupTestGitRepo(t *testing.T, baseDir string) string {
	repoDir := filepath.Join(baseDir, "test-repo")
	err := os.MkdirAll(repoDir, 0o755)
	require.NoError(t, err)

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = repoDir
	require.NoError(t, cmd.Run())

	// Configure git
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = repoDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = repoDir
	require.NoError(t, cmd.Run())

	// Create compose file
	composeContent := `services:
  nginx:
    image: nginx:alpine
    ports:
      - "8080:80"
`
	composeFile := filepath.Join(repoDir, "compose.yaml")
	err = os.WriteFile(composeFile, []byte(composeContent), 0o644)
	require.NoError(t, err)

	// Commit files
	cmd = exec.Command("git", "add", ".")
	cmd.Dir = repoDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = repoDir
	require.NoError(t, cmd.Run())

	return repoDir
}
