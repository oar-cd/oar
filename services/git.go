package services

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/go-git/go-git/v5"
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

// Clone clones a repository with optional authentication
func (s *GitService) Clone(gitURL string, gitAuth *GitAuthConfig, workingDir string) error {
	slog.Info("Cloning repository", "git_url", gitURL, "working_dir", workingDir)

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

	_, err = git.PlainCloneContext(ctx, workingDir, false, cloneOptions)
	if err != nil {
		slog.Error("Service operation failed",
			"layer", "git",
			"operation", "git_clone",
			"git_url", gitURL,
			"working_dir", workingDir,
			"error", err)
		return err
	}

	slog.Info("Repository cloned successfully", "git_url", gitURL, "working_dir", workingDir)
	return nil
}

// Pull pulls latest changes from remote with optional authentication
func (s *GitService) Pull(gitAuth *GitAuthConfig, workingDir string) error {
	slog.Debug("Pulling repository changes", "working_dir", workingDir)

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

	err = worktree.PullContext(ctx, pullOptions)
	if err != nil && err != git.NoErrAlreadyUpToDate {
		slog.Error("Service operation failed",
			"layer", "git",
			"operation", "git_pull",
			"working_dir", workingDir,
			"error", err)
		return err
	}

	if err == git.NoErrAlreadyUpToDate {
		slog.Debug("Repository already up to date", "working_dir", workingDir)
	} else {
		slog.Info("Repository changes pulled successfully", "working_dir", workingDir)
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
