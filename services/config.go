package services

import (
	"os"
	"path/filepath"
)

const (
	DataDir     = ".oar"
	ProjectsDir = "projects"
	GitDir      = "git"
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
		dataDir = filepath.Join(homeDir, DataDir)
	}

	return &Config{
		DataDir:      dataDir,
		WorkspaceDir: filepath.Join(dataDir, ProjectsDir),
	}
}
