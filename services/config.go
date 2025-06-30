package services

import (
	"os"
	"path/filepath"
)

// Config holds configuration for all services
type Config struct {
	DataDir      string
	WorkspaceDir string
}

// NewConfig creates a new configuration with sensible defaults
func NewConfig(dataDir string) *Config {
	if dataDir == "" {
		homeDir, _ := os.UserHomeDir()
		dataDir = filepath.Join(homeDir, ".oar")
	}

	return &Config{
		DataDir:      dataDir,
		WorkspaceDir: filepath.Join(dataDir, "projects"),
	}
}

// ProjectWorkingDir returns the working directory for a specific project
func (c *Config) ProjectWorkingDir(projectID string) string {
	return filepath.Join(c.WorkspaceDir, projectID)
}

