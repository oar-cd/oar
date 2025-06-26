package services

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/go-git/go-git/v5"
)

type GitService struct{}

func NewGitService() GitService {
	return GitService{}
}

// Clone clones a repository
func (s *GitService) Clone(gitURL, workingDir string) error {
	slog.Info("Cloning repository", "git_url", gitURL, "working_dir", workingDir)

	cloneOptions := &git.CloneOptions{
		URL:          gitURL,
		SingleBranch: true,
	}

	_, err := git.PlainClone(workingDir, false, cloneOptions)
	if err != nil {
		slog.Error("Failed to clone repository", "git_url", gitURL, "error", err)
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	slog.Info("Repository cloned successfully", "git_url", gitURL, "working_dir", workingDir)
	return nil
}

// Pull pulls latest changes from remote
func (s *GitService) Pull(workingDir string) error {
	slog.Debug("Pulling repository changes", "working_dir", workingDir)

	repo, err := git.PlainOpen(workingDir)
	if err != nil {
		slog.Error("Failed to open repository", "working_dir", workingDir, "error", err)
		return fmt.Errorf("failed to open repository: %w", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		slog.Error("Failed to get worktree", "working_dir", workingDir, "error", err)
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	pullOptions := &git.PullOptions{
		SingleBranch: true,
	}

	err = worktree.Pull(pullOptions)
	if err != nil && err != git.NoErrAlreadyUpToDate {
		slog.Error("Failed to pull changes", "working_dir", workingDir, "error", err)
		return fmt.Errorf("failed to pull: %w", err)
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
		return "", fmt.Errorf("failed to open repository: %w", err)
	}

	ref, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD: %w", err)
	}

	return ref.Hash().String(), nil
}

// GetCommitInfo returns commit information
func (s GitService) GetCommitInfo(workingDir string) (*CommitInfo, error) {
	repo, err := git.PlainOpen(workingDir)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	ref, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}

	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to get commit: %w", err)
	}

	return &CommitInfo{
		Hash:    commit.Hash.String(),
		Message: commit.Message,
		Author:  commit.Author.Name,
		Date:    commit.Author.When,
	}, nil
}

type CommitInfo struct {
	Hash    string
	Message string
	Author  string
	Date    time.Time
}
