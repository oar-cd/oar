package services

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

// GitAuthConfig holds Git authentication configuration for a project
type GitAuthConfig struct {
	HTTPAuth *GitHTTPAuthConfig
	SSHAuth  *GitSSHAuthConfig
}

// GitHTTPAuthConfig for HTTP basic authentication (GitHub tokens, etc.)
type GitHTTPAuthConfig struct {
	Username string // "token" for GitHub
	Password string // actual token/password
}

// GitSSHAuthConfig for passwordless SSH key authentication
type GitSSHAuthConfig struct {
	PrivateKey string // PEM-encoded private key as string
	User       string // SSH user (default: "git")
}

// GitAuthType represents the Git authentication method type
type GitAuthType string

const (
	GitAuthTypeHTTP GitAuthType = "http"
	GitAuthTypeSSH  GitAuthType = "ssh"
)

// String implements the Stringer interface
func (a GitAuthType) String() string {
	return string(a)
}

// IsValid checks if the GitAuthType is valid
func (a GitAuthType) IsValid() bool {
	switch a {
	case GitAuthTypeHTTP, GitAuthTypeSSH:
		return true
	default:
		return false
	}
}

// ParseGitAuthType parses a string into a GitAuthType
func ParseGitAuthType(s string) (GitAuthType, error) {
	authType := GitAuthType(s)
	if !authType.IsValid() {
		return "", fmt.Errorf("invalid auth type: %s", s)
	}
	return authType, nil
}

type GitService struct {
	config *Config
}

// Ensure GitService implements GitExecutor
var _ GitExecutor = (*GitService)(nil)

func NewGitService(config *Config) *GitService {
	return &GitService{
		config: config,
	}
}

// createAuthMethod creates a transport.AuthMethod from GitAuthConfig
func (s *GitService) createAuthMethod(auth *GitAuthConfig) (transport.AuthMethod, error) {
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
func (s *GitService) createSSHAuth(config *GitSSHAuthConfig) (transport.AuthMethod, error) {
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
func (s *GitService) Clone(gitURL string, gitBranch string, gitAuth *GitAuthConfig, workingDir string) error {
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

	_, err = git.PlainCloneContext(ctx, workingDir, false, cloneOptions)
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
func (s *GitService) Pull(gitBranch string, gitAuth *GitAuthConfig, workingDir string) error {
	slog.Debug("Pulling repository changes", "git_branch", gitBranch, "working_dir", workingDir)

	// Create authentication method
	authMethod, err := s.createAuthMethod(gitAuth)
	if err != nil {
		slog.Error("Service operation failed",
			"layer", "git",
			"operation", "git_pull_auth",
			"working_dir", workingDir,
			"error", err)
		return fmt.Errorf("failed to create auth method: %w", err)
	}

	repo, err := git.PlainOpen(workingDir)
	if err != nil {
		slog.Error("Service operation failed",
			"layer", "git",
			"operation", "git_pull",
			"working_dir", workingDir,
			"error", err)
		return err
	}

	worktree, err := repo.Worktree()
	if err != nil {
		slog.Error("Service operation failed",
			"layer", "git",
			"operation", "git_pull",
			"working_dir", workingDir,
			"error", err)
		return err
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), s.config.GitTimeout)
	defer cancel()

	pullOptions := &git.PullOptions{
		SingleBranch: true,
		Auth:         authMethod,
	}

	// If a specific branch is provided, set the reference name
	if gitBranch != "" {
		pullOptions.ReferenceName = plumbing.NewBranchReferenceName(gitBranch)
	}

	err = worktree.PullContext(ctx, pullOptions)
	if err != nil && err != git.NoErrAlreadyUpToDate {
		slog.Error("Service operation failed",
			"layer", "git",
			"operation", "git_pull",
			"git_branch", gitBranch,
			"working_dir", workingDir,
			"error", err)
		return err
	}

	if err == git.NoErrAlreadyUpToDate {
		slog.Debug("Repository already up to date", "git_branch", gitBranch, "working_dir", workingDir)
	} else {
		slog.Info("Repository changes pulled successfully", "git_branch", gitBranch, "working_dir", workingDir)
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
func (s *GitService) Fetch(gitBranch string, gitAuth *GitAuthConfig, workingDir string) error {
	slog.Debug("Fetching from Git repository", "git_branch", gitBranch, "working_dir", workingDir)

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
	}

	// If a specific branch is provided, fetch only that branch
	if gitBranch != "" {
		fetchOptions.RefSpecs = []config.RefSpec{
			config.RefSpec(fmt.Sprintf("+refs/heads/%s:refs/remotes/origin/%s", gitBranch, gitBranch)),
		}
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
func (s *GitService) TestAuthentication(gitURL string, gitAuth *GitAuthConfig) error {
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
func (s *GitService) testGitLsRemote(gitURL string, gitAuth *GitAuthConfig) error {
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
func (s *GitService) GetDefaultBranch(gitURL string, gitAuth *GitAuthConfig) (string, error) {
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
						slog.Info("Found default branch via commit hash", "git_url", gitURL, "default_branch", branchName)
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
