package git_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClone_PackFilePermissions(t *testing.T) {
	tempDir := t.TempDir()
	bareRepo := filepath.Join(tempDir, "bare.git")
	workRepo := filepath.Join(tempDir, "work")
	cloneRepo := filepath.Join(tempDir, "clone")

	createBareRepo(t, bareRepo)
	initRepoWithCommit(t, workRepo, map[string]string{
		"file1.txt": "content1",
		"file2.txt": "content2",
		"file3.txt": "content3",
	})
	addRemote(t, workRepo, "origin", bareRepo)
	pushToRemote(t, workRepo, "main")

	addCommitToRepo(t, workRepo, map[string]string{
		"file4.txt": "content4",
		"file5.txt": "content5",
	})
	pushToRemote(t, workRepo, "main")

	addCommitToRepo(t, workRepo, map[string]string{
		"file6.txt": "content6",
		"file7.txt": "content7",
	})
	pushToRemote(t, workRepo, "main")

	gitService := setupGitService(t)
	err := gitService.Clone(bareRepo, "main", nil, cloneRepo)
	require.NoError(t, err, "Clone should succeed")

	packDir := filepath.Join(cloneRepo, ".git", "objects", "pack")
	entries, err := os.ReadDir(packDir)
	require.NoError(t, err, "Should be able to read pack directory")

	var packFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".pack" {
			packFiles = append(packFiles, entry.Name())
		}
	}

	require.NotEmpty(t, packFiles, "Should have at least one pack file")

	for _, packFile := range packFiles {
		packPath := filepath.Join(packDir, packFile)
		info, err := os.Stat(packPath)
		require.NoError(t, err, "Should be able to stat pack file %s", packFile)

		mode := info.Mode()
		perm := mode.Perm()
		require.Equal(t, os.FileMode(0o644), perm, "Pack file %s should have 0644 permissions", packFile)
	}
}
