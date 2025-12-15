package git_test

import (
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/oar-cd/oar/config"
	"github.com/oar-cd/oar/git"
	"github.com/oar-cd/oar/logging"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	logging.InitLogging("debug")

	// Configure git identity globally for all test operations
	// This ensures that repos created via shell git commands in tests
	// have the necessary config even if they don't explicitly set it
	// These settings are critical in CI environments where git config may not be initialized
	if err := exec.Command("git", "config", "--global", "user.name", "Test User").Run(); err != nil {
		slog.Warn("Failed to set git user.name", "error", err)
	}
	if err := exec.Command("git", "config", "--global", "user.email", "test@example.com").Run(); err != nil {
		slog.Warn("Failed to set git user.email", "error", err)
	}
	if err := exec.Command("git", "config", "--global", "init.defaultBranch", "main").Run(); err != nil {
		slog.Warn("Failed to set git init.defaultBranch", "error", err)
	}

	exitCode := m.Run()

	// Clean up global git config after tests
	// Ignore errors during cleanup as these settings may not exist or may be locked
	_ = exec.Command("git", "config", "--global", "--unset", "user.name").Run()
	_ = exec.Command("git", "config", "--global", "--unset", "user.email").Run()
	_ = exec.Command("git", "config", "--global", "--unset", "init.defaultBranch").Run()

	os.Exit(exitCode)
}

func setupGitService(t *testing.T) *git.GitService {
	cfg := &config.Config{
		GitTimeout: 30 * time.Second,
	}
	return git.NewGitService(cfg)
}

func createBareRepo(t *testing.T, dir string) {
	err := os.MkdirAll(dir, 0o755)
	require.NoError(t, err, "Failed to create bare repo directory")

	cmd := exec.Command("git", "init", "--bare")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to init bare repository: %s", string(output))
}

func initRepoWithCommit(t *testing.T, dir string, files map[string]string) {
	err := os.MkdirAll(dir, 0o755)
	require.NoError(t, err, "Failed to create repo directory")

	cmd := exec.Command("git", "init", "-b", "main")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to init repository: %s", string(output))

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = dir
	output, err = cmd.CombinedOutput()
	require.NoError(t, err, "Failed to set git user.name: %s", string(output))

	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = dir
	output, err = cmd.CombinedOutput()
	require.NoError(t, err, "Failed to set git user.email: %s", string(output))

	for path, content := range files {
		fullPath := filepath.Join(dir, path)
		dirPath := filepath.Dir(fullPath)
		err = os.MkdirAll(dirPath, 0o755)
		require.NoError(t, err, "Failed to create directory for file %s", path)

		err = os.WriteFile(fullPath, []byte(content), 0o644)
		require.NoError(t, err, "Failed to write file %s", path)
	}

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = dir
	output, err = cmd.CombinedOutput()
	require.NoError(t, err, "Failed to stage files: %s", string(output))

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = dir
	output, err = cmd.CombinedOutput()
	require.NoError(t, err, "Failed to commit files: %s", string(output))
}

func cloneRepo(t *testing.T, source, dest string, branch string) {
	args := []string{"clone", source, dest}
	if branch != "" {
		args = append(args, "-b", branch)
	}
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to clone repository: %s", string(output))
}

func addCommitToRepo(t *testing.T, dir string, files map[string]string) {
	for path, content := range files {
		fullPath := filepath.Join(dir, path)
		dirPath := filepath.Dir(fullPath)
		err := os.MkdirAll(dirPath, 0o755)
		require.NoError(t, err, "Failed to create directory for file %s", path)

		err = os.WriteFile(fullPath, []byte(content), 0o644)
		require.NoError(t, err, "Failed to write file %s", path)
	}

	cmd := exec.Command("git", "add", ".")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to stage files: %s", string(output))

	cmd = exec.Command("git", "commit", "-m", "Add files")
	cmd.Dir = dir
	output, err = cmd.CombinedOutput()
	require.NoError(t, err, "Failed to commit files: %s", string(output))
}

func createUntrackedFile(t *testing.T, dir, path, content string) {
	fullPath := filepath.Join(dir, path)
	dirPath := filepath.Dir(fullPath)
	err := os.MkdirAll(dirPath, 0o755)
	require.NoError(t, err, "Failed to create directory for untracked file %s", path)

	err = os.WriteFile(fullPath, []byte(content), 0o644)
	require.NoError(t, err, "Failed to write untracked file %s", path)
}

func modifyTrackedFile(t *testing.T, dir, path, content string) {
	fullPath := filepath.Join(dir, path)
	err := os.WriteFile(fullPath, []byte(content), 0o644)
	require.NoError(t, err, "Failed to modify tracked file %s", path)
}

func stageFile(t *testing.T, dir, path string) {
	cmd := exec.Command("git", "add", path)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to stage file %s: %s", path, string(output))
}

func getCommitHash(t *testing.T, dir string) string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to get commit hash: %s", string(output))
	return strings.TrimSpace(string(output))
}

func fileExists(t *testing.T, dir, path string) bool {
	fullPath := filepath.Join(dir, path)
	_, err := os.Stat(fullPath)
	return err == nil
}

func fileContent(t *testing.T, dir, path string) string {
	fullPath := filepath.Join(dir, path)
	content, err := os.ReadFile(fullPath)
	require.NoError(t, err, "Failed to read file %s", path)
	return string(content)
}

func checkoutCommit(t *testing.T, dir, commitHash string) {
	cmd := exec.Command("git", "checkout", commitHash)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to checkout commit %s: %s", commitHash, string(output))
}

func createBranch(t *testing.T, dir, branchName string) {
	cmd := exec.Command("git", "branch", branchName)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to create branch %s: %s", branchName, string(output))
}

func checkoutBranch(t *testing.T, dir, branchName string) {
	cmd := exec.Command("git", "checkout", branchName)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to checkout branch %s: %s", branchName, string(output))
}

func getCurrentBranch(t *testing.T, dir string) string {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to get current branch: %s", string(output))
	return strings.TrimSpace(string(output))
}

func isDetachedHead(t *testing.T, dir string) bool {
	branch := getCurrentBranch(t, dir)
	return branch == "HEAD"
}

func addRemote(t *testing.T, dir, remoteName, remoteURL string) {
	cmd := exec.Command("git", "remote", "add", remoteName, remoteURL)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to add remote %s: %s", remoteName, string(output))
}

func pushToRemote(t *testing.T, dir string, branch string) {
	cmd := exec.Command("git", "push", "origin", branch)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to push to remote: %s", string(output))
}

func deleteFile(t *testing.T, dir, path string) {
	fullPath := filepath.Join(dir, path)
	err := os.Remove(fullPath)
	require.NoError(t, err, "Failed to delete file %s", path)
}

func renameFile(t *testing.T, dir, oldPath, newPath string) {
	oldFullPath := filepath.Join(dir, oldPath)
	newFullPath := filepath.Join(dir, newPath)
	cmd := exec.Command("git", "mv", oldPath, newPath)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to rename file %s to %s: %s", oldPath, newPath, string(output))

	_, err = os.Stat(oldFullPath)
	require.True(t, os.IsNotExist(err), "Old file should not exist")
	_, err = os.Stat(newFullPath)
	require.NoError(t, err, "New file should exist")
}

func setupTestRepos(t *testing.T, initialFiles map[string]string) (bareRepo, workRepo, localRepo string) {
	tempDir := t.TempDir()
	bareRepo = filepath.Join(tempDir, "bare.git")
	workRepo = filepath.Join(tempDir, "work")
	localRepo = filepath.Join(tempDir, "local")

	createBareRepo(t, bareRepo)
	initRepoWithCommit(t, workRepo, initialFiles)
	addRemote(t, workRepo, "origin", bareRepo)
	pushToRemote(t, workRepo, "main")
	cloneRepo(t, bareRepo, localRepo, "")

	return bareRepo, workRepo, localRepo
}

func TestPull_NoRemoteChanges(t *testing.T) {
	_, _, localRepo := setupTestRepos(t, map[string]string{
		"file1.txt": "content1",
	})

	gitService := setupGitService(t)
	initialHash := getCommitHash(t, localRepo)

	err := gitService.Pull("main", nil, localRepo)
	require.NoError(t, err, "Pull should succeed with no changes")

	finalHash := getCommitHash(t, localRepo)
	require.Equal(t, initialHash, finalHash, "Commit hash should remain the same")
	require.True(t, fileExists(t, localRepo, "file1.txt"), "file1.txt should exist")
	require.Equal(t, "content1", fileContent(t, localRepo, "file1.txt"), "file1.txt content should match")
}

func TestPull_RemoteHasChanges(t *testing.T) {
	tempDir := t.TempDir()
	bareRepo := filepath.Join(tempDir, "bare.git")
	workRepo := filepath.Join(tempDir, "work")
	localRepo := filepath.Join(tempDir, "local")

	createBareRepo(t, bareRepo)
	initRepoWithCommit(t, workRepo, map[string]string{
		"file1.txt": "content1",
	})
	addRemote(t, workRepo, "origin", bareRepo)
	pushToRemote(t, workRepo, "main")
	cloneRepo(t, bareRepo, localRepo, "")

	initialHash := getCommitHash(t, localRepo)

	addCommitToRepo(t, workRepo, map[string]string{
		"file2.txt": "content2",
	})
	pushToRemote(t, workRepo, "main")

	gitService := setupGitService(t)
	err := gitService.Pull("main", nil, localRepo)
	require.NoError(t, err, "Pull should succeed with remote changes")

	finalHash := getCommitHash(t, localRepo)
	require.NotEqual(t, initialHash, finalHash, "Commit hash should change")
	require.Equal(t, getCommitHash(t, workRepo), finalHash, "Commit hash should match remote")
	require.True(t, fileExists(t, localRepo, "file1.txt"), "file1.txt should exist")
	require.True(t, fileExists(t, localRepo, "file2.txt"), "file2.txt should exist")
	require.Equal(t, "content2", fileContent(t, localRepo, "file2.txt"), "file2.txt content should match")
}

func TestPull_NoRemoteChanges_WithUntrackedFiles(t *testing.T) {
	tempDir := t.TempDir()
	bareRepo := filepath.Join(tempDir, "bare.git")
	workRepo := filepath.Join(tempDir, "work")
	localRepo := filepath.Join(tempDir, "local")

	createBareRepo(t, bareRepo)
	initRepoWithCommit(t, workRepo, map[string]string{
		"file1.txt": "content1",
	})
	addRemote(t, workRepo, "origin", bareRepo)
	pushToRemote(t, workRepo, "main")
	cloneRepo(t, bareRepo, localRepo, "")

	createUntrackedFile(t, localRepo, "untracked.txt", "untracked content")

	gitService := setupGitService(t)
	err := gitService.Pull("main", nil, localRepo)
	require.NoError(t, err, "Pull should succeed with untracked files")

	require.True(t, fileExists(t, localRepo, "untracked.txt"), "untracked.txt should still exist")
	require.Equal(
		t,
		"untracked content",
		fileContent(t, localRepo, "untracked.txt"),
		"untracked.txt content should be preserved",
	)
}

func TestPull_RemoteHasChanges_WithUntrackedFiles(t *testing.T) {
	tempDir := t.TempDir()
	bareRepo := filepath.Join(tempDir, "bare.git")
	workRepo := filepath.Join(tempDir, "work")
	localRepo := filepath.Join(tempDir, "local")

	createBareRepo(t, bareRepo)
	initRepoWithCommit(t, workRepo, map[string]string{
		"file1.txt": "content1",
	})
	addRemote(t, workRepo, "origin", bareRepo)
	pushToRemote(t, workRepo, "main")
	cloneRepo(t, bareRepo, localRepo, "")

	createUntrackedFile(t, localRepo, "untracked.txt", "untracked content")

	addCommitToRepo(t, workRepo, map[string]string{
		"file2.txt": "content2",
	})
	pushToRemote(t, workRepo, "main")

	gitService := setupGitService(t)
	err := gitService.Pull("main", nil, localRepo)
	require.NoError(t, err, "Pull should succeed with remote changes and untracked files")

	require.True(t, fileExists(t, localRepo, "untracked.txt"), "untracked.txt should still exist")
	require.Equal(
		t,
		"untracked content",
		fileContent(t, localRepo, "untracked.txt"),
		"untracked.txt content should be preserved",
	)
	require.True(t, fileExists(t, localRepo, "file2.txt"), "file2.txt should exist from remote")
}

func TestPull_NoRemoteChanges_WithTrackedFilesChanged(t *testing.T) {
	tempDir := t.TempDir()
	bareRepo := filepath.Join(tempDir, "bare.git")
	workRepo := filepath.Join(tempDir, "work")
	localRepo := filepath.Join(tempDir, "local")

	createBareRepo(t, bareRepo)
	initRepoWithCommit(t, workRepo, map[string]string{
		"file1.txt": "content1",
	})
	addRemote(t, workRepo, "origin", bareRepo)
	pushToRemote(t, workRepo, "main")
	cloneRepo(t, bareRepo, localRepo, "")

	modifyTrackedFile(t, localRepo, "file1.txt", "modified content")

	gitService := setupGitService(t)
	err := gitService.Pull("main", nil, localRepo)
	require.Error(t, err, "Pull should fail when tracked files are modified")
	require.Contains(
		t,
		err.Error(),
		"cannot pull with modified tracked files",
		"Error should indicate modified tracked files",
	)

	require.Equal(
		t,
		"modified content",
		fileContent(t, localRepo, "file1.txt"),
		"file1.txt should remain modified",
	)
}

func TestPull_RemoteHasChanges_WithTrackedFilesChanged(t *testing.T) {
	tempDir := t.TempDir()
	bareRepo := filepath.Join(tempDir, "bare.git")
	workRepo := filepath.Join(tempDir, "work")
	localRepo := filepath.Join(tempDir, "local")

	createBareRepo(t, bareRepo)
	initRepoWithCommit(t, workRepo, map[string]string{
		"file1.txt": "content1",
		"file2.txt": "content2",
	})
	addRemote(t, workRepo, "origin", bareRepo)
	pushToRemote(t, workRepo, "main")
	cloneRepo(t, bareRepo, localRepo, "")

	modifyTrackedFile(t, localRepo, "file1.txt", "local modification")

	addCommitToRepo(t, workRepo, map[string]string{
		"file3.txt": "content3",
	})
	pushToRemote(t, workRepo, "main")

	gitService := setupGitService(t)
	err := gitService.Pull("main", nil, localRepo)
	require.Error(t, err, "Pull should fail when tracked files are modified")
	require.Contains(
		t,
		err.Error(),
		"cannot pull with modified tracked files",
		"Error should indicate modified tracked files",
	)

	require.Equal(t, "local modification", fileContent(t, localRepo, "file1.txt"), "file1.txt should remain modified")
	require.False(t, fileExists(t, localRepo, "file3.txt"), "file3.txt should not be pulled due to error")
}

func TestPull_RemoteAndLocalModifySameFile(t *testing.T) {
	tempDir := t.TempDir()
	bareRepo := filepath.Join(tempDir, "bare.git")
	workRepo := filepath.Join(tempDir, "work")
	localRepo := filepath.Join(tempDir, "local")

	createBareRepo(t, bareRepo)
	initRepoWithCommit(t, workRepo, map[string]string{
		"file1.txt": "content1",
	})
	addRemote(t, workRepo, "origin", bareRepo)
	pushToRemote(t, workRepo, "main")
	cloneRepo(t, bareRepo, localRepo, "")

	modifyTrackedFile(t, localRepo, "file1.txt", "local modification")

	modifyTrackedFile(t, workRepo, "file1.txt", "remote modification")
	addCommitToRepo(t, workRepo, map[string]string{})
	pushToRemote(t, workRepo, "main")

	gitService := setupGitService(t)
	err := gitService.Pull("main", nil, localRepo)
	require.Error(t, err, "Pull should fail when tracked files are modified")
	require.Contains(
		t,
		err.Error(),
		"cannot pull with modified tracked files",
		"Error should indicate modified tracked files",
	)

	require.Equal(t, "local modification", fileContent(t, localRepo, "file1.txt"), "Local changes should remain")
}

func TestPull_RemoteDeletesFile(t *testing.T) {
	tempDir := t.TempDir()
	bareRepo := filepath.Join(tempDir, "bare.git")
	workRepo := filepath.Join(tempDir, "work")
	localRepo := filepath.Join(tempDir, "local")

	createBareRepo(t, bareRepo)
	initRepoWithCommit(t, workRepo, map[string]string{
		"file1.txt": "content1",
		"file2.txt": "content2",
	})
	addRemote(t, workRepo, "origin", bareRepo)
	pushToRemote(t, workRepo, "main")
	cloneRepo(t, bareRepo, localRepo, "")

	deleteFile(t, workRepo, "file2.txt")
	addCommitToRepo(t, workRepo, map[string]string{})
	pushToRemote(t, workRepo, "main")

	gitService := setupGitService(t)
	err := gitService.Pull("main", nil, localRepo)
	require.NoError(t, err, "Pull should succeed with file deletion")

	require.True(t, fileExists(t, localRepo, "file1.txt"), "file1.txt should still exist")
	require.False(t, fileExists(t, localRepo, "file2.txt"), "file2.txt should be deleted")
}

func TestPull_RemoteRenamesFile(t *testing.T) {
	tempDir := t.TempDir()
	bareRepo := filepath.Join(tempDir, "bare.git")
	workRepo := filepath.Join(tempDir, "work")
	localRepo := filepath.Join(tempDir, "local")

	createBareRepo(t, bareRepo)
	initRepoWithCommit(t, workRepo, map[string]string{
		"oldname.txt": "content",
	})
	addRemote(t, workRepo, "origin", bareRepo)
	pushToRemote(t, workRepo, "main")
	cloneRepo(t, bareRepo, localRepo, "")

	renameFile(t, workRepo, "oldname.txt", "newname.txt")
	addCommitToRepo(t, workRepo, map[string]string{})
	pushToRemote(t, workRepo, "main")

	gitService := setupGitService(t)
	err := gitService.Pull("main", nil, localRepo)
	require.NoError(t, err, "Pull should succeed with file rename")

	require.False(t, fileExists(t, localRepo, "oldname.txt"), "oldname.txt should not exist")
	require.True(t, fileExists(t, localRepo, "newname.txt"), "newname.txt should exist")
	require.Equal(t, "content", fileContent(t, localRepo, "newname.txt"), "Content should be preserved")
}

func TestPull_MultipleCommitsAhead(t *testing.T) {
	tempDir := t.TempDir()
	bareRepo := filepath.Join(tempDir, "bare.git")
	workRepo := filepath.Join(tempDir, "work")
	localRepo := filepath.Join(tempDir, "local")

	createBareRepo(t, bareRepo)
	initRepoWithCommit(t, workRepo, map[string]string{
		"file1.txt": "content1",
	})
	addRemote(t, workRepo, "origin", bareRepo)
	pushToRemote(t, workRepo, "main")
	cloneRepo(t, bareRepo, localRepo, "")

	addCommitToRepo(t, workRepo, map[string]string{
		"file2.txt": "content2",
	})
	pushToRemote(t, workRepo, "main")

	addCommitToRepo(t, workRepo, map[string]string{
		"file3.txt": "content3",
	})
	pushToRemote(t, workRepo, "main")

	addCommitToRepo(t, workRepo, map[string]string{
		"file4.txt": "content4",
	})
	pushToRemote(t, workRepo, "main")

	gitService := setupGitService(t)
	err := gitService.Pull("main", nil, localRepo)
	require.NoError(t, err, "Pull should succeed with multiple commits")

	require.Equal(t, getCommitHash(t, workRepo), getCommitHash(t, localRepo), "Should be at latest commit")
	require.True(t, fileExists(t, localRepo, "file2.txt"), "file2.txt should exist")
	require.True(t, fileExists(t, localRepo, "file3.txt"), "file3.txt should exist")
	require.True(t, fileExists(t, localRepo, "file4.txt"), "file4.txt should exist")
}

func TestPull_WithStagedChanges(t *testing.T) {
	tempDir := t.TempDir()
	bareRepo := filepath.Join(tempDir, "bare.git")
	workRepo := filepath.Join(tempDir, "work")
	localRepo := filepath.Join(tempDir, "local")

	createBareRepo(t, bareRepo)
	initRepoWithCommit(t, workRepo, map[string]string{
		"file1.txt": "content1",
	})
	addRemote(t, workRepo, "origin", bareRepo)
	pushToRemote(t, workRepo, "main")
	cloneRepo(t, bareRepo, localRepo, "")

	modifyTrackedFile(t, localRepo, "file1.txt", "modified")
	stageFile(t, localRepo, "file1.txt")

	gitService := setupGitService(t)
	err := gitService.Pull("main", nil, localRepo)
	require.Error(t, err, "Pull should fail with staged changes")
	require.Contains(
		t,
		err.Error(),
		"cannot pull with modified tracked files",
		"Error should indicate modified tracked files",
	)

	require.Equal(t, "modified", fileContent(t, localRepo, "file1.txt"), "Staged changes should remain")
}

func TestPull_WithDivergedHistory(t *testing.T) {
	tempDir := t.TempDir()
	bareRepo := filepath.Join(tempDir, "bare.git")
	workRepo := filepath.Join(tempDir, "work")
	localRepo := filepath.Join(tempDir, "local")

	createBareRepo(t, bareRepo)
	initRepoWithCommit(t, workRepo, map[string]string{
		"file1.txt": "content1",
	})
	addRemote(t, workRepo, "origin", bareRepo)
	pushToRemote(t, workRepo, "main")
	cloneRepo(t, bareRepo, localRepo, "")

	addCommitToRepo(t, localRepo, map[string]string{
		"local-only.txt": "local commit",
	})

	addCommitToRepo(t, workRepo, map[string]string{
		"remote-only.txt": "remote commit",
	})
	pushToRemote(t, workRepo, "main")

	gitService := setupGitService(t)
	err := gitService.Pull("main", nil, localRepo)
	require.NoError(t, err, "Pull should succeed with diverged history")

	require.Equal(t, getCommitHash(t, workRepo), getCommitHash(t, localRepo), "Should match remote commit")
	require.True(t, fileExists(t, localRepo, "remote-only.txt"), "remote-only.txt should exist")
	require.False(t, fileExists(t, localRepo, "local-only.txt"), "local-only.txt should be discarded")
}

func TestPull_AfterForcePush(t *testing.T) {
	tempDir := t.TempDir()
	bareRepo := filepath.Join(tempDir, "bare.git")
	workRepo := filepath.Join(tempDir, "work")
	localRepo := filepath.Join(tempDir, "local")

	createBareRepo(t, bareRepo)
	initRepoWithCommit(t, workRepo, map[string]string{
		"file1.txt": "content1",
	})
	addRemote(t, workRepo, "origin", bareRepo)
	pushToRemote(t, workRepo, "main")

	addCommitToRepo(t, workRepo, map[string]string{
		"file2.txt": "content2",
	})
	pushToRemote(t, workRepo, "main")

	cloneRepo(t, bareRepo, localRepo, "")

	cmd := exec.Command("git", "reset", "--hard", "HEAD~1")
	cmd.Dir = workRepo
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to reset HEAD~1: %s", string(output))

	addCommitToRepo(t, workRepo, map[string]string{
		"file3.txt": "content3",
	})

	cmd = exec.Command("git", "push", "--force", "origin", "main")
	cmd.Dir = workRepo
	output, err = cmd.CombinedOutput()
	require.NoError(t, err, "Failed to force push: %s", string(output))

	gitService := setupGitService(t)
	err = gitService.Pull("main", nil, localRepo)
	require.NoError(t, err, "Pull should succeed after force push")

	require.Equal(t, getCommitHash(t, workRepo), getCommitHash(t, localRepo), "Should match new remote history")
	require.True(t, fileExists(t, localRepo, "file3.txt"), "file3.txt should exist")
	require.False(t, fileExists(t, localRepo, "file2.txt"), "file2.txt should not exist")
}

func TestPull_UntrackedFileConflictsWithIncoming(t *testing.T) {
	tempDir := t.TempDir()
	bareRepo := filepath.Join(tempDir, "bare.git")
	workRepo := filepath.Join(tempDir, "work")
	localRepo := filepath.Join(tempDir, "local")

	createBareRepo(t, bareRepo)
	initRepoWithCommit(t, workRepo, map[string]string{
		"file1.txt": "content1",
	})
	addRemote(t, workRepo, "origin", bareRepo)
	pushToRemote(t, workRepo, "main")
	cloneRepo(t, bareRepo, localRepo, "")

	createUntrackedFile(t, localRepo, "newfile.txt", "local untracked content")

	addCommitToRepo(t, workRepo, map[string]string{
		"newfile.txt": "remote tracked content",
	})
	pushToRemote(t, workRepo, "main")

	gitService := setupGitService(t)
	err := gitService.Pull("main", nil, localRepo)
	require.Error(t, err, "Pull should fail when untracked file conflicts with incoming tracked file")
	require.Contains(
		t,
		err.Error(),
		"untracked files would be overwritten",
		"Error should indicate untracked file conflict",
	)
}

func TestPull_FromDetachedHead(t *testing.T) {
	tempDir := t.TempDir()
	bareRepo := filepath.Join(tempDir, "bare.git")
	workRepo := filepath.Join(tempDir, "work")
	localRepo := filepath.Join(tempDir, "local")

	createBareRepo(t, bareRepo)
	initRepoWithCommit(t, workRepo, map[string]string{
		"file1.txt": "content1",
	})
	addRemote(t, workRepo, "origin", bareRepo)
	pushToRemote(t, workRepo, "main")
	cloneRepo(t, bareRepo, localRepo, "")

	commitHash := getCommitHash(t, localRepo)
	checkoutCommit(t, localRepo, commitHash)

	require.True(t, isDetachedHead(t, localRepo), "Should be in detached HEAD state")

	addCommitToRepo(t, workRepo, map[string]string{
		"file2.txt": "content2",
	})
	pushToRemote(t, workRepo, "main")

	gitService := setupGitService(t)
	err := gitService.Pull("main", nil, localRepo)
	require.NoError(t, err, "Pull should succeed from detached HEAD")

	require.Equal(t, getCommitHash(t, workRepo), getCommitHash(t, localRepo), "Should be at latest commit")
	require.True(t, fileExists(t, localRepo, "file2.txt"), "file2.txt should exist")
}

func TestPull_FromNonDefaultBranch(t *testing.T) {
	tempDir := t.TempDir()
	bareRepo := filepath.Join(tempDir, "bare.git")
	workRepo := filepath.Join(tempDir, "work")
	localRepo := filepath.Join(tempDir, "local")

	createBareRepo(t, bareRepo)
	initRepoWithCommit(t, workRepo, map[string]string{
		"file1.txt": "content1",
	})
	addRemote(t, workRepo, "origin", bareRepo)
	pushToRemote(t, workRepo, "main")

	createBranch(t, workRepo, "feature")
	checkoutBranch(t, workRepo, "feature")
	addCommitToRepo(t, workRepo, map[string]string{
		"feature.txt": "feature content",
	})
	pushToRemote(t, workRepo, "feature")
	checkoutBranch(t, workRepo, "main")

	cloneRepo(t, bareRepo, localRepo, "")

	cmd := exec.Command("git", "fetch", "origin", "feature:feature")
	cmd.Dir = localRepo
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to fetch feature branch: %s", string(output))

	checkoutBranch(t, localRepo, "feature")
	require.Equal(t, "feature", getCurrentBranch(t, localRepo), "Should be on feature branch")

	addCommitToRepo(t, workRepo, map[string]string{
		"file2.txt": "content2",
	})
	pushToRemote(t, workRepo, "main")

	gitService := setupGitService(t)
	err = gitService.Pull("main", nil, localRepo)
	require.NoError(t, err, "Pull should succeed from non-default branch")

	require.Equal(t, getCommitHash(t, workRepo), getCommitHash(t, localRepo), "Should be at latest main commit")
	require.True(t, fileExists(t, localRepo, "file2.txt"), "file2.txt should exist")
	require.False(t, fileExists(t, localRepo, "feature.txt"), "feature.txt should not exist after switching to main")
}

func TestPull_FromNonDefaultBranch_WithUntrackedFiles(t *testing.T) {
	tempDir := t.TempDir()
	bareRepo := filepath.Join(tempDir, "bare.git")
	workRepo := filepath.Join(tempDir, "work")
	localRepo := filepath.Join(tempDir, "local")

	createBareRepo(t, bareRepo)
	initRepoWithCommit(t, workRepo, map[string]string{
		"file1.txt": "content1",
	})
	addRemote(t, workRepo, "origin", bareRepo)
	pushToRemote(t, workRepo, "main")

	createBranch(t, workRepo, "feature")
	checkoutBranch(t, workRepo, "feature")
	addCommitToRepo(t, workRepo, map[string]string{
		"feature.txt": "feature content",
	})
	pushToRemote(t, workRepo, "feature")
	checkoutBranch(t, workRepo, "main")

	cloneRepo(t, bareRepo, localRepo, "")

	cmd := exec.Command("git", "fetch", "origin", "feature:feature")
	cmd.Dir = localRepo
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to fetch feature branch: %s", string(output))

	checkoutBranch(t, localRepo, "feature")
	createUntrackedFile(t, localRepo, "untracked.txt", "untracked content")

	gitService := setupGitService(t)
	err = gitService.Pull("main", nil, localRepo)
	require.NoError(t, err, "Pull should succeed from non-default branch with untracked files")

	require.True(t, fileExists(t, localRepo, "untracked.txt"), "untracked.txt should be preserved")
	require.Equal(
		t,
		"untracked content",
		fileContent(t, localRepo, "untracked.txt"),
		"untracked.txt content should be preserved",
	)
}

func TestPull_FromNonDefaultBranch_WithChangedTrackedFiles(t *testing.T) {
	tempDir := t.TempDir()
	bareRepo := filepath.Join(tempDir, "bare.git")
	workRepo := filepath.Join(tempDir, "work")
	localRepo := filepath.Join(tempDir, "local")

	createBareRepo(t, bareRepo)
	initRepoWithCommit(t, workRepo, map[string]string{
		"file1.txt": "content1",
	})
	addRemote(t, workRepo, "origin", bareRepo)
	pushToRemote(t, workRepo, "main")

	createBranch(t, workRepo, "feature")
	checkoutBranch(t, workRepo, "feature")
	addCommitToRepo(t, workRepo, map[string]string{
		"feature.txt": "feature content",
	})
	pushToRemote(t, workRepo, "feature")
	checkoutBranch(t, workRepo, "main")

	cloneRepo(t, bareRepo, localRepo, "")

	cmd := exec.Command("git", "fetch", "origin", "feature:feature")
	cmd.Dir = localRepo
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to fetch feature branch: %s", string(output))

	checkoutBranch(t, localRepo, "feature")
	modifyTrackedFile(t, localRepo, "feature.txt", "modified feature content")

	gitService := setupGitService(t)
	err = gitService.Pull("main", nil, localRepo)
	require.Error(t, err, "Pull should fail when tracked files are modified")
	require.Contains(
		t,
		err.Error(),
		"cannot pull with modified tracked files",
		"Error should indicate modified tracked files",
	)

	require.True(t, fileExists(t, localRepo, "feature.txt"), "feature.txt should still exist on feature branch")
	require.Equal(
		t,
		"modified feature content",
		fileContent(t, localRepo, "feature.txt"),
		"feature.txt should remain modified",
	)
}

func TestPull_WhenTrackedBranchDeletedOnRemote(t *testing.T) {
	tempDir := t.TempDir()
	bareRepo := filepath.Join(tempDir, "bare.git")
	workRepo := filepath.Join(tempDir, "work")
	localRepo := filepath.Join(tempDir, "local")

	createBareRepo(t, bareRepo)
	initRepoWithCommit(t, workRepo, map[string]string{
		"file1.txt": "content1",
	})
	addRemote(t, workRepo, "origin", bareRepo)
	pushToRemote(t, workRepo, "main")

	createBranch(t, workRepo, "feature")
	checkoutBranch(t, workRepo, "feature")
	addCommitToRepo(t, workRepo, map[string]string{
		"feature.txt": "feature content",
	})
	pushToRemote(t, workRepo, "feature")
	checkoutBranch(t, workRepo, "main")

	cloneRepo(t, bareRepo, localRepo, "feature")

	cmd := exec.Command("git", "push", "origin", "--delete", "feature")
	cmd.Dir = workRepo
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to delete remote branch: %s", string(output))

	gitService := setupGitService(t)
	err = gitService.Pull("feature", nil, localRepo)
	require.Error(t, err, "Pull should fail when remote branch is deleted")
}

func TestPull_WithUncommittedDeletionOfTrackedFile(t *testing.T) {
	tempDir := t.TempDir()
	bareRepo := filepath.Join(tempDir, "bare.git")
	workRepo := filepath.Join(tempDir, "work")
	localRepo := filepath.Join(tempDir, "local")

	createBareRepo(t, bareRepo)
	initRepoWithCommit(t, workRepo, map[string]string{
		"file1.txt": "content1",
		"file2.txt": "content2",
	})
	addRemote(t, workRepo, "origin", bareRepo)
	pushToRemote(t, workRepo, "main")
	cloneRepo(t, bareRepo, localRepo, "")

	deleteFile(t, localRepo, "file2.txt")

	gitService := setupGitService(t)
	err := gitService.Pull("main", nil, localRepo)
	require.Error(t, err, "Pull should fail with uncommitted deletion")
	require.Contains(
		t,
		err.Error(),
		"cannot pull with modified tracked files",
		"Error should indicate modified tracked files",
	)

	require.False(t, fileExists(t, localRepo, "file2.txt"), "Deleted file should remain deleted")
}
