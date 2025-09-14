package project

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oar-cd/oar/app"
	"github.com/oar-cd/oar/cmd/test"
	"github.com/oar-cd/oar/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testOptions struct {
	gitURL string
	name   string
}

func TestProjectAddHappy(t *testing.T) {
	tests := []struct {
		name           string
		options        testOptions
		composeFiles   []string
		expectedOutput string
		expectError    string
	}{
		{
			name: "Success",
			options: testOptions{
				gitURL: "",
				name:   "test-project",
			},
			composeFiles:   []string{"testdata/composefiles/compose.yaml"},
			expectedOutput: "testdata/output/project_add.golden",
			expectError:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for test data
			tempDir, err := os.MkdirTemp("", "oar-test-*")
			require.NoError(t, err)
			defer func() {
				if err := os.RemoveAll(tempDir); err != nil {
					t.Logf("Warning: failed to clean up temp directory: %v", err)
				}
			}()

			// Setup a test git repository
			repoDir := filepath.Join(tempDir, "test-git-repo")

			var repoFiles []test.RepoFile

			for _, file := range tt.composeFiles {
				_, fileName := filepath.Split(file)
				fileContent, err := test.ReadFile(file)
				require.NoError(t, err, "Failed to read compose file %s", file)

				repoFiles = append(repoFiles, test.RepoFile{
					Path:    fileName,
					Content: fileContent,
				})
			}

			_, err = test.InitGitRepo(repoDir, repoFiles)
			assert.NoError(t, err, "Failed to initialize git repository")

			// Set encryption key for testing
			t.Setenv("OAR_ENCRYPTION_KEY", "cw_0x689RpI-jtRR7oE8h_eQsKImvJapLeSbXpwF4e4=") // Test key

			// Initialize the app with test data directory
			t.Setenv("OAR_DATA_DIR", tempDir)
			config, err := services.NewConfig("", services.WithCLIDefaults())
			require.NoError(t, err)
			err = app.InitializeWithConfig(config)
			require.NoError(t, err)

			// Build args
			args := []string{"--git-url", repoDir, "--name", tt.options.name, "--compose-file", "docker-compose.yml"}

			var stdout, stderr bytes.Buffer
			rootCmd := NewCmdProjectAdd()
			rootCmd.SetOut(&stdout)
			rootCmd.SetErr(&stderr)
			rootCmd.SetArgs(args)

			// Debug: Print the command being executed
			t.Logf("Executing command: oar %s", strings.Join(args, " "))

			err = rootCmd.Execute()

			stdoutStr := test.Trim(stdout.String())
			stderrStr := stderr.String()

			if err != nil {
				t.Logf("Command execution failed: %s", err)
				t.Logf("Standard Output: %s", stdoutStr)
				t.Logf("Standard Error: %s", stderrStr)
				t.FailNow()
			}

			// Find the created project by listing all projects and finding by name
			projects, err := app.GetProjectService().List()
			if err != nil {
				t.Fatalf("Failed to list projects: %v", err)
			}

			var createdProject *services.Project
			for _, p := range projects {
				if p.Name == tt.options.name {
					createdProject = p
					break
				}
			}
			if createdProject == nil {
				t.Fatalf("Created project with name %s not found in project list", tt.options.name)
			}

			// Verify results
			assert.NoError(t, err, "Expected command to succeed")
			data := map[string]any{
				"ProjectID":     createdProject.ID.String(),
				"ProjectName":   createdProject.Name,
				"ProjectsDir":   filepath.Join(tempDir, services.ProjectsDir),
				"ProjectGitURL": createdProject.GitURL,
			}
			expectedOutput := test.RenderTemplate(data)
			t.Logf("Expected output: %s", expectedOutput)
			t.Logf("Actual output: %s", stdoutStr)
			assert.Equal(t, expectedOutput, stdoutStr, "Output should match expected format")
		})
	}
}

func TestProjectAddValidation(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		expectedError string
	}{
		{
			name: "Empty name should fail",
			args: []string{
				"--git-url",
				"https://github.com/test/repo.git",
				"--name",
				"",
				"--compose-file",
				"docker-compose.yml",
			},
			expectedError: "name is required",
		},
		{
			name: "Whitespace-only name should fail",
			args: []string{
				"--git-url",
				"https://github.com/test/repo.git",
				"--name",
				"   \t\n   ",
				"--compose-file",
				"docker-compose.yml",
			},
			expectedError: "name is required",
		},
		{
			name:          "Empty git URL should fail",
			args:          []string{"--git-url", "", "--name", "test-project", "--compose-file", "docker-compose.yml"},
			expectedError: "git URL is required",
		},
		{
			name: "Whitespace-only git URL should fail",
			args: []string{
				"--git-url",
				"   \t\n   ",
				"--name",
				"test-project",
				"--compose-file",
				"docker-compose.yml",
			},
			expectedError: "git URL is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for test data
			tempDir, err := os.MkdirTemp("", "oar-test-validation-*")
			require.NoError(t, err)
			defer func() {
				if err := os.RemoveAll(tempDir); err != nil {
					t.Logf("Warning: failed to clean up temp directory: %v", err)
				}
			}()

			// Set encryption key for testing
			t.Setenv("OAR_ENCRYPTION_KEY", "cw_0x689RpI-jtRR7oE8h_eQsKImvJapLeSbXpwF4e4=") // Test key

			// Initialize the app with test data directory
			t.Setenv("OAR_DATA_DIR", tempDir)
			config, err := services.NewConfig("", services.WithCLIDefaults())
			require.NoError(t, err)
			err = app.InitializeWithConfig(config)
			require.NoError(t, err)

			var stdout, stderr bytes.Buffer
			rootCmd := NewCmdProjectAdd()
			rootCmd.SetOut(&stdout)
			rootCmd.SetErr(&stderr)
			rootCmd.SetArgs(tt.args)

			// Execute command and expect it to fail
			err = rootCmd.Execute()

			// Verify command failed with expected error
			assert.Error(t, err, "Expected command to fail")
			assert.Contains(
				t,
				err.Error(),
				tt.expectedError,
				"Error message should contain expected validation message",
			)
		})
	}
}
