// Package domain provides core domain types and entities for Oar.
package domain

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

const (
	// GitDir is the directory name for Git repositories within a project's working directory
	GitDir = "git"
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

type Project struct {
	ID              uuid.UUID
	Name            string
	GitURL          string
	GitBranch       string         // Git branch to use (never empty, always set to default branch if not specified)
	GitAuth         *GitAuthConfig // Git authentication configuration
	WorkingDir      string
	ComposeFiles    []string
	ComposeOverride *string  // Optional Docker Compose override content
	Variables       []string // Variables in .env format, one per string
	Status          ProjectStatus
	LastCommit      *string
	WatcherEnabled  bool // Enable automatic deployments on git changes
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (p *Project) GitDir() (string, error) {
	if p.WorkingDir == "" {
		return "", fmt.Errorf("working directory is not set for project %s", p.Name)
	}
	return filepath.Join(p.WorkingDir, GitDir), nil
}

func (p *Project) LastCommitStr() string {
	if p.LastCommit == nil {
		return ""
	}
	return *p.LastCommit
}

// GetDeletedDirectoryPath calculates the path where a project directory will be moved when deleted
func GetDeletedDirectoryPath(workingDir string) string {
	deletedDirName := fmt.Sprintf("deleted-%s", filepath.Base(workingDir))
	return filepath.Join(filepath.Dir(workingDir), deletedDirName)
}

func NewProject(name, gitURL string, composeFiles []string, variables []string) Project {
	return Project{
		ID:             uuid.New(),
		Name:           name,
		GitURL:         gitURL,
		GitBranch:      "", // Default to repository's default branch
		ComposeFiles:   composeFiles,
		Variables:      variables,
		Status:         ProjectStatusStopped,
		WatcherEnabled: true, // Default to enabled
	}
}
