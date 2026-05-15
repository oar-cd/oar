// Package git provides Git repository operations including clone, pull, and authentication.
package git

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/config"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"
	"github.com/go-git/go-git/v6/plumbing/transport"
	"github.com/go-git/go-git/v6/plumbing/transport/http"
	"github.com/go-git/go-git/v6/plumbing/transport/ssh"
	oarconfig "github.com/oar-cd/oar/config"
	"github.com/oar-cd/oar/domain"
)

type GitService struct {
	config *oarconfig.Config
}

func NewGitService(config *oarconfig.Config) *GitService {
	return &GitService{
		config: config,
	}
}

// createAuthMethod creates a transport.AuthMethod from GitAuthConfig
func (s *GitService) createAuthMethod(auth *domain.GitAuthConfig) (transport.AuthMethod, error) {
	if auth == nil {
		return nil, nil // Public repo
	}

	// HTTP authentication (GitHub tokens, etc.)
	if auth.HTTPAuth != nil {
		return &http.BasicAuth{
			Username: auth.HTTPAuth.Username,
			Password: auth.HTTPAuth.Password,
		}, nil
	}

	// SSH key authentication
	if auth.SSHAuth != nil {
		return s.createSSHAuth(auth.SSHAuth)
	}

	// Neither auth method configured = public repo
	return nil, nil
}

// createSSHAuth creates SSH authentication from SSHAuthConfig
func (s *GitService) createSSHAuth(config *domain.GitSSHAuthConfig) (transport.AuthMethod, error) {
	if config == nil {
		return nil, fmt.Errorf("SSH auth config is nil")
	}

	user := config.User
	if user == "" {
		user = "git" // Default for Git operations
	}

	// Use NewPublicKeys with key bytes directly (passwordless)
	keyBytes := []byte(config.PrivateKey)
	return ssh.NewPublicKeys(user, keyBytes, "") // Empty password for passwordless keys
}

// Clone clones a repository with optional authentication and branch
func (s *GitService) Clone(gitURL string, gitBranch string, gitAuth *domain.GitAuthConfig, workingDir string) error {
	slog.Info("Cloning repository", "git_url", gitURL, "git_branch", gitBranch, "working_dir", workingDir)

	// Create authentication method
	authMethod, err := s.createAuthMethod(gitAuth)
	if err != nil {
		slog.Error("Service operation failed",
			"layer", "git",
			"operation", "git_clone_auth",
			"git_url", gitURL,
			"working_dir", workingDir,
			"error", err)
		return fmt.Errorf("failed to create auth method: %w", err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), s.config.GitTimeout)
	defer cancel()

	cloneOptions := &git.CloneOptions{
		URL:          gitURL,
		SingleBranch: true,
		Auth:         authMethod,
	}

	// If a specific branch is requested, set it in clone options
	if gitBranch != "" {
		cloneOptions.ReferenceName = plumbing.NewBranchReferenceName(gitBranch)
	}

	_, err = git.PlainCloneContext(ctx, workingDir, cloneOptions)
	if err != nil {
		slog.Error("Service operation failed",
			"layer", "git",
			"operation", "git_clone",
			"git_url", gitURL,
			"git_branch", gitBranch,
			"working_dir", workingDir,
			"error", err)
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	slog.Info("Repository cloned successfully", "git_url", gitURL, "git_branch", gitBranch, "working_dir", workingDir)
	return nil
}

// Pull pulls latest changes from remote with optional authentication
// Uses fetch + reset approach to handle force-pushes (equivalent to git fetch && git reset --hard origin/branch)
func (s *GitService) Pull(gitBranch string, gitAuth *domain.GitAuthConfig, workingDir string) error {
	slog.Debug("Pulling repository changes", "git_branch", gitBranch, "working_dir", workingDir)

	// First, fetch the latest changes from remote
	err := s.Fetch(gitBranch, gitAuth, workingDir)
	if err != nil {
		slog.Error("Service operation failed",
			"layer", "git",
			"operation", "git_pull_fetch",
			"git_branch", gitBranch,
			"working_dir", workingDir,
			"error", err)
		return fmt.Errorf("failed to fetch changes: %w", err)
	}

	// Open the repository
	repo, err := git.PlainOpen(workingDir)
	if err != nil {
		slog.Error("Service operation failed",
			"layer", "git",
			"operation", "git_pull",
			"working_dir", workingDir,
			"error", err)
		return err
	}

	// Get the worktree
	worktree, err := repo.Worktree()
	if err != nil {
		slog.Error("Service operation failed",
			"layer", "git",
			"operation", "git_pull",
			"working_dir", workingDir,
			"error", err)
		return err
	}

	// Branch name is required - should never be empty in production
	if gitBranch == "" {
		slog.Error("Service operation failed",
			"layer", "git",
			"operation", "git_pull",
			"working_dir", workingDir,
			"error", "git branch is required")
		return fmt.Errorf("git branch is required")
	}
	branchName := gitBranch

	// Get current commit hash
	currentCommit, err := s.GetLatestCommit(workingDir)
	if err != nil {
		// Continue anyway, this is just for logging
		currentCommit = "unknown"
	}

	// Get the remote reference
	remoteBranchName := fmt.Sprintf("refs/remotes/origin/%s", branchName)
	ref, err := repo.Reference(plumbing.ReferenceName(remoteBranchName), true)
	if err != nil {
		slog.Error("Service operation failed",
			"layer", "git",
			"operation", "git_pull_get_remote_ref",
			"git_branch", branchName,
			"working_dir", workingDir,
			"remote_ref", remoteBranchName,
			"error", err)
		return fmt.Errorf("failed to get remote reference %s: %w", remoteBranchName, err)
	}

	// Check if we're already up to date with the remote commit
	upToDate := currentCommit == ref.Hash().String()

	// Check for modified tracked files first (before checking untracked conflicts)
	err = s.checkModifiedTrackedFiles(worktree)
	if err != nil {
		slog.Error("Service operation failed",
			"layer", "git",
			"operation", "git_pull_check_modified_files",
			"git_branch", branchName,
			"working_dir", workingDir,
			"error", err)
		return err
	}

	// Check for conflicts between untracked files and incoming tracked files
	err = s.checkUntrackedConflicts(repo, worktree, ref.Hash())
	if err != nil {
		slog.Error("Service operation failed",
			"layer", "git",
			"operation", "git_pull_check_conflicts",
			"git_branch", branchName,
			"working_dir", workingDir,
			"target_commit", ref.Hash().String(),
			"error", err)
		return err
	}

	// Reset worktree to match remote commit exactly
	// At this point we've verified there are no modified tracked files and no conflicting untracked files
	// We use MergeReset which, with the fix in PR #1540, preserves untracked files
	err = worktree.Reset(&git.ResetOptions{
		Mode:   git.MergeReset,
		Commit: ref.Hash(),
	})
	if err != nil {
		slog.Error("Service operation failed",
			"layer", "git",
			"operation", "git_pull_reset",
			"git_branch", branchName,
			"working_dir", workingDir,
			"target_commit", ref.Hash().String(),
			"error", err)
		return fmt.Errorf("failed to reset to %s: %w", ref.Hash().String(), err)
	}

	if upToDate {
		slog.Debug("Working directory synchronized with remote", "git_branch", branchName, "working_dir", workingDir)
	} else {
		slog.Info("Repository updated successfully",
			"git_branch", branchName,
			"working_dir", workingDir,
			"from_commit", currentCommit,
			"to_commit", ref.Hash().String())
	}

	return nil
}

// GetLatestCommit returns the latest commit hash
func (s *GitService) GetLatestCommit(workingDir string) (string, error) {
	repo, err := git.PlainOpen(workingDir)
	if err != nil {
		slog.Error("Service operation failed",
			"layer", "git",
			"operation", "git_get_commit",
			"working_dir", workingDir,
			"error", err)
		return "", err
	}

	ref, err := repo.Head()
	if err != nil {
		slog.Error("Service operation failed",
			"layer", "git",
			"operation", "git_get_commit",
			"working_dir", workingDir,
			"error", err)
		return "", err
	}

	return ref.Hash().String(), nil
}

// Fetch fetches the latest changes from remote without merging
func (s *GitService) Fetch(gitBranch string, gitAuth *domain.GitAuthConfig, workingDir string) error {
	slog.Debug("Fetching from Git repository", "git_branch", gitBranch, "working_dir", workingDir)

	// Branch name is required - should never be empty in production
	if gitBranch == "" {
		slog.Error("Service operation failed",
			"layer", "git",
			"operation", "git_fetch",
			"working_dir", workingDir,
			"error", "git branch is required")
		return fmt.Errorf("git branch is required")
	}

	repo, err := git.PlainOpen(workingDir)
	if err != nil {
		slog.Error("Service operation failed",
			"layer", "git",
			"operation", "git_fetch",
			"git_branch", gitBranch,
			"working_dir", workingDir,
			"error", err)
		return err
	}

	authMethod, err := s.createAuthMethod(gitAuth)
	if err != nil {
		return fmt.Errorf("failed to create auth method: %w", err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), s.config.GitTimeout)
	defer cancel()

	fetchOptions := &git.FetchOptions{
		Auth: authMethod,
		RefSpecs: []config.RefSpec{
			config.RefSpec(fmt.Sprintf("+refs/heads/%s:refs/remotes/origin/%s", gitBranch, gitBranch)),
		},
	}

	err = repo.FetchContext(ctx, fetchOptions)
	if err != nil && err != git.NoErrAlreadyUpToDate {
		slog.Error("Service operation failed",
			"layer", "git",
			"operation", "git_fetch",
			"git_branch", gitBranch,
			"working_dir", workingDir,
			"error", err)
		return err
	}

	if err == git.NoErrAlreadyUpToDate {
		slog.Debug("Repository already up to date", "git_branch", gitBranch, "working_dir", workingDir)
	} else {
		slog.Info("Repository fetched successfully", "git_branch", gitBranch, "working_dir", workingDir)
	}

	return nil
}

// GetRemoteLatestCommit returns the latest commit hash from the remote branch
func (s *GitService) GetRemoteLatestCommit(workingDir string, gitBranch string) (string, error) {
	// Branch name is required - should never be empty in production
	if gitBranch == "" {
		slog.Error("Service operation failed",
			"layer", "git",
			"operation", "git_get_remote_commit",
			"working_dir", workingDir,
			"error", "git branch is required")
		return "", fmt.Errorf("git branch is required")
	}

	repo, err := git.PlainOpen(workingDir)
	if err != nil {
		slog.Error("Service operation failed",
			"layer", "git",
			"operation", "git_get_remote_commit",
			"git_branch", gitBranch,
			"working_dir", workingDir,
			"error", err)
		return "", err
	}

	// Get remote reference for the branch
	remoteBranchName := fmt.Sprintf("refs/remotes/origin/%s", gitBranch)
	ref, err := repo.Reference(plumbing.ReferenceName(remoteBranchName), true)
	if err != nil {
		slog.Error("Service operation failed",
			"layer", "git",
			"operation", "git_get_remote_commit",
			"git_branch", gitBranch,
			"working_dir", workingDir,
			"remote_ref", remoteBranchName,
			"error", err)
		return "", err
	}

	return ref.Hash().String(), nil
}

// TestAuthentication tests Git authentication using ls-remote operation
// This is more resistant to credential caching than clone operations
func (s *GitService) TestAuthentication(gitURL string, gitAuth *domain.GitAuthConfig) error {
	slog.Info("Testing Git authentication", "git_url", gitURL)

	// Use ls-remote operation instead of clone to avoid credential caching
	err := s.testGitLsRemote(gitURL, gitAuth)
	if err != nil {
		slog.Error("Git authentication test failed",
			"layer", "service",
			"operation", "test_git_authentication",
			"git_url", gitURL,
			"error", err)
		return err
	}

	slog.Info("Git authentication test successful", "git_url", gitURL)
	return nil
}

// testGitLsRemote performs a git ls-remote operation to test authentication
func (s *GitService) testGitLsRemote(gitURL string, gitAuth *domain.GitAuthConfig) error {
	// Create authentication method using existing logic
	authMethod, err := s.createAuthMethod(gitAuth)
	if err != nil {
		return fmt.Errorf("failed to create auth method: %w", err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), s.config.GitTimeout)
	defer cancel()

	// Use git ls-remote to test authentication
	remote := git.NewRemote(nil, &config.RemoteConfig{
		Name: "origin",
		URLs: []string{gitURL},
	})

	_, err = remote.ListContext(ctx, &git.ListOptions{
		Auth: authMethod,
	})

	return err
}

// GetDefaultBranch determines the default branch of a remote Git repository
func (s *GitService) GetDefaultBranch(gitURL string, gitAuth *domain.GitAuthConfig) (string, error) {
	slog.Info("Getting default branch", "git_url", gitURL)

	// Create authentication method
	authMethod, err := s.createAuthMethod(gitAuth)
	if err != nil {
		return "", fmt.Errorf("failed to create auth method: %w", err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), s.config.GitTimeout)
	defer cancel()

	// Use git ls-remote to list remote references
	remote := git.NewRemote(nil, &config.RemoteConfig{
		Name: "origin",
		URLs: []string{gitURL},
	})

	refs, err := remote.ListContext(ctx, &git.ListOptions{
		Auth: authMethod,
	})
	if err != nil {
		slog.Error("Service operation failed",
			"layer", "git",
			"operation", "get_default_branch",
			"git_url", gitURL,
			"error", err)
		return "", fmt.Errorf("failed to list remote references: %w", err)
	}

	// Look for HEAD reference to find default branch
	for _, ref := range refs {
		if ref.Name() == plumbing.HEAD {
			// Check if this is a symbolic reference (HEAD -> refs/heads/main)
			if ref.Type() == plumbing.SymbolicReference {
				// Get the target reference name and extract branch name
				target := ref.Target()
				if target.IsBranch() {
					branchName := target.Short()
					slog.Info(
						"Found default branch via symbolic reference",
						"git_url",
						gitURL,
						"default_branch",
						branchName,
					)
					return branchName, nil
				}
			} else {
				// HEAD points directly to a commit, find which branch has the same hash
				for _, otherRef := range refs {
					if otherRef.Hash() == ref.Hash() && otherRef.Name().IsBranch() {
						branchName := otherRef.Name().Short()
						slog.Info("Found default branch via commit hash",
							"git_url", gitURL,
							"default_branch", branchName)
						return branchName, nil
					}
				}
			}
		}
	}

	// Could not determine default branch
	slog.Error("Could not determine default branch", "git_url", gitURL)
	return "", fmt.Errorf("could not determine default branch for repository %s", gitURL)
}

// checkUntrackedConflicts checks if any untracked files would conflict with incoming tracked files
func (s *GitService) checkUntrackedConflicts(
	repo *git.Repository,
	worktree *git.Worktree,
	targetHash plumbing.Hash,
) error {
	// Get current worktree status to find untracked files
	status, err := worktree.Status()
	if err != nil {
		return fmt.Errorf("failed to get worktree status: %w", err)
	}

	// Collect untracked files
	untrackedFiles := make(map[string]bool)
	for file, fileStatus := range status {
		if fileStatus.Worktree == git.Untracked {
			untrackedFiles[file] = true
		}
	}

	// If no untracked files, no conflicts possible
	if len(untrackedFiles) == 0 {
		return nil
	}

	// Get target commit to check which files it contains
	targetCommit, err := repo.CommitObject(targetHash)
	if err != nil {
		return fmt.Errorf("failed to get target commit: %w", err)
	}

	targetTree, err := targetCommit.Tree()
	if err != nil {
		return fmt.Errorf("failed to get target tree: %w", err)
	}

	// Check if any untracked files exist in the target commit
	var conflictingFiles []string
	err = targetTree.Files().ForEach(func(f *object.File) error {
		if untrackedFiles[f.Name] {
			conflictingFiles = append(conflictingFiles, f.Name)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to iterate target tree files: %w", err)
	}

	// If there are conflicts, return an error
	if len(conflictingFiles) > 0 {
		return fmt.Errorf("untracked files would be overwritten by checkout: %v", conflictingFiles)
	}

	return nil
}

// checkModifiedTrackedFiles checks if any tracked files have been modified (staged or unstaged)
func (s *GitService) checkModifiedTrackedFiles(worktree *git.Worktree) error {
	// Get current worktree status
	status, err := worktree.Status()
	if err != nil {
		return fmt.Errorf("failed to get worktree status: %w", err)
	}

	// Collect modified tracked files
	var modifiedFiles []string
	for file, fileStatus := range status {
		// Skip untracked files
		if fileStatus.Worktree == git.Untracked {
			continue
		}

		// Check if file is modified (either in worktree or staging area)
		if fileStatus.Worktree != git.Unmodified || fileStatus.Staging != git.Unmodified {
			modifiedFiles = append(modifiedFiles, file)
		}
	}

	// If there are modified tracked files, return an error
	if len(modifiedFiles) > 0 {
		return fmt.Errorf("cannot pull with modified tracked files: %v", modifiedFiles)
	}

	return nil
}
