package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadFile(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expectError bool
	}{
		{
			name:        "simple text file",
			content:     "hello world",
			expectError: false,
		},
		{
			name:        "multi-line file",
			content:     "line 1\nline 2\nline 3",
			expectError: false,
		},
		{
			name:        "empty file",
			content:     "",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.txt")

			err := os.WriteFile(tmpFile, []byte(tt.content), 0o644)
			require.NoError(t, err)

			// Test ReadFile function
			result, err := ReadFile(tmpFile)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.content, result)
			}
		})
	}
}

func TestReadFile_NonExistentFile(t *testing.T) {
	result, err := ReadFile("/non/existent/file.txt")
	assert.Error(t, err)
	assert.Empty(t, result)
}

func TestTrim(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single line with trailing spaces",
			input:    "hello world   ",
			expected: "hello world",
		},
		{
			name:     "multiple lines with trailing spaces",
			input:    "line1   \nline2 \t \nline3\n",
			expected: "line1\nline2 \t\nline3\n",
		},
		{
			name:     "no trailing spaces",
			input:    "clean line\nanother clean line",
			expected: "clean line\nanother clean line",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only whitespace",
			input:    "   \t  \n  \t ",
			expected: "   \t\n  \t",
		},
		{
			name:     "preserve internal spaces",
			input:    "word1   word2   \nword3 word4  ",
			expected: "word1   word2\nword3 word4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Trim(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestInitGitRepo(t *testing.T) {
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "test-repo")

	files := []RepoFile{
		{Path: "README.md", Content: "# Test Repository"},
		{Path: "main.go", Content: "package main\n\nfunc main() {}"},
	}

	repo, err := InitGitRepo(repoPath, files)
	require.NoError(t, err)
	require.NotNil(t, repo)

	// Verify repository was created
	_, err = os.Stat(filepath.Join(repoPath, ".git"))
	assert.NoError(t, err, "Git repository should be initialized")

	// Verify files were created
	for _, file := range files {
		content, err := os.ReadFile(filepath.Join(repoPath, file.Path))
		assert.NoError(t, err)
		assert.Equal(t, file.Content, string(content))
	}

	// Verify git log has initial commit
	_, err = repo.Worktree()
	require.NoError(t, err)

	head, err := repo.Head()
	require.NoError(t, err)

	commit, err := repo.CommitObject(head.Hash())
	require.NoError(t, err)
	assert.Equal(t, "Initial commit", commit.Message)
	assert.Equal(t, "John Doe", commit.Author.Name)
	assert.Equal(t, "john@doe.org", commit.Author.Email)
}

func TestAddRepoFiles(t *testing.T) {
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "test-repo")

	// Initialize empty git repo
	repo, err := git.PlainInit(repoPath, false)
	require.NoError(t, err)

	worktree, err := repo.Worktree()
	require.NoError(t, err)

	files := []RepoFile{
		{Path: "file1.txt", Content: "content 1"},
		{Path: "file2.txt", Content: "content 2"},
	}

	err = AddRepoFiles(worktree, files)
	require.NoError(t, err)

	// Verify files were created and added to git
	for _, file := range files {
		// Check file exists
		content, err := os.ReadFile(filepath.Join(repoPath, file.Path))
		assert.NoError(t, err)
		assert.Equal(t, file.Content, string(content))

		// Check file is staged
		status, err := worktree.Status()
		require.NoError(t, err)
		fileStatus := status.File(file.Path)
		assert.Equal(t, git.Added, fileStatus.Staging)
	}
}

func TestRenderTemplate_Integration(t *testing.T) {
	// This test requires the golden file to exist, so we'll create a simple version
	tmpDir := t.TempDir()
	testdataDir := filepath.Join(tmpDir, "testdata", "output")
	err := os.MkdirAll(testdataDir, 0o755)
	require.NoError(t, err)

	goldenFile := filepath.Join(testdataDir, "project_add.golden")
	templateContent := "Project: {{.name}}\nURL: {{.url}}"
	err = os.WriteFile(goldenFile, []byte(templateContent), 0o644)
	require.NoError(t, err)

	// Change to the temp directory for the test
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	defer func() {
		if err := os.Chdir(originalWd); err != nil {
			t.Logf("Warning: failed to restore working directory: %v", err)
		}
	}()

	data := map[string]any{
		"name": "test-project",
		"url":  "https://github.com/test/repo",
	}

	result := RenderTemplate(data)
	expected := "Project: test-project\nURL: https://github.com/test/repo"
	assert.Equal(t, expected, result)
}
