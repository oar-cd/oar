package services

import (
	"context"
	"log/slog"

	"github.com/go-git/go-git/v5"
)

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

// Clone clones a repository
func (s *GitService) Clone(gitURL, workingDir string) error {
	slog.Info("Cloning repository", "git_url", gitURL, "working_dir", workingDir)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), s.config.GitTimeout)
	defer cancel()

	cloneOptions := &git.CloneOptions{
		URL:          gitURL,
		SingleBranch: true,
	}

	_, err := git.PlainCloneContext(ctx, workingDir, false, cloneOptions)
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

// Pull pulls latest changes from remote
func (s *GitService) Pull(workingDir string) error {
	slog.Debug("Pulling repository changes", "working_dir", workingDir)

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
