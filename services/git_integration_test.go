package services

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGitService_Pull_ForcePush tests that pull handles force-pushes correctly
func TestGitService_Pull_ForcePush(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create temporary directories for "remote" and "local" repositories
	tempDir := t.TempDir()
	remoteDir := filepath.Join(tempDir, "remote")
	localDir := filepath.Join(tempDir, "local")

	// Initialize the "remote" repository
	_, err := git.PlainInit(remoteDir, true) // bare repository
	require.NoError(t, err)

	// Create a working repository to push to remote
	workingDir := filepath.Join(tempDir, "working")
	workingRepo, err := git.PlainInit(workingDir, false)
	require.NoError(t, err)

	// Configure the working repository
	worktree, err := workingRepo.Worktree()
	require.NoError(t, err)

	// Create initial commit
	testFile := filepath.Join(workingDir, "test.txt")
	err = os.WriteFile(testFile, []byte("initial content"), 0644)
	require.NoError(t, err)

	_, err = worktree.Add("test.txt")
	require.NoError(t, err)

	initialCommit, err := worktree.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	require.NoError(t, err)

	// Add remote and push
	_, err = workingRepo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{remoteDir},
	})
	require.NoError(t, err)

	// Push to remote
	err = workingRepo.Push(&git.PushOptions{})
	require.NoError(t, err)

	// Set up the GitService
	config := &Config{
		GitTimeout: 30 * time.Second,
	}
	gitService := NewGitService(config)

	// Clone the remote repository to local directory with specific branch
	err = gitService.Clone(remoteDir, "master", nil, localDir)
	require.NoError(t, err)

	// Verify initial state
	localCommit, err := gitService.GetLatestCommit(localDir)
	require.NoError(t, err)
	assert.Equal(t, initialCommit.String(), localCommit)

	// Now simulate a force-push scenario:
	// First, make a local change in the cloned repository to create divergence
	localTestFile := filepath.Join(localDir, "test.txt")
	err = os.WriteFile(localTestFile, []byte("local changes"), 0644)
	require.NoError(t, err)

	localRepo, err := git.PlainOpen(localDir)
	require.NoError(t, err)

	localWorktree, err := localRepo.Worktree()
	require.NoError(t, err)

	_, err = localWorktree.Add("test.txt")
	require.NoError(t, err)

	_, err = localWorktree.Commit("Local commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Local User",
			Email: "local@example.com",
			When:  time.Now(),
		},
	})
	require.NoError(t, err)

	// In the working repository, create a different commit from the same initial point
	err = os.WriteFile(testFile, []byte("different remote content"), 0644)
	require.NoError(t, err)

	_, err = worktree.Add("test.txt")
	require.NoError(t, err)

	differentCommit, err := worktree.Commit("Different remote commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Remote User",
			Email: "remote@example.com",
			When:  time.Now(),
		},
	})
	require.NoError(t, err)

	// Force push the different commit (simulating a force-push that rewrites history)
	err = workingRepo.Push(&git.PushOptions{
		Force: true,
	})
	require.NoError(t, err)

	// Now try to pull in the local repository
	// This should handle the force-push scenario gracefully
	err = gitService.Pull("master", nil, localDir)

	// The pull should succeed (our new implementation should handle force-pushes)
	assert.NoError(t, err, "Pull should succeed even with force-pushed changes")

	// Verify that local repository now has the force-pushed commit
	newLocalCommit, err := gitService.GetLatestCommit(localDir)
	require.NoError(t, err)
	assert.Equal(t, differentCommit.String(), newLocalCommit, "Local commit should match the force-pushed commit")

	// Verify the content was updated
	content, err := os.ReadFile(filepath.Join(localDir, "test.txt"))
	require.NoError(t, err)
	assert.Equal(t, "different remote content", string(content), "File content should match force-pushed version")
}
