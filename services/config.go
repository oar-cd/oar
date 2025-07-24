package services

import (
	"os"
	"path/filepath"
)

const (
	DataDir     = ".oar"
	ProjectsDir = "projects"
	GitDir      = "git"
	TmpDir      = "tmp"
)

// Config holds configuration for all services
type Config struct {
	// DataDir is the directory where all data is stored
	DataDir string
	// TmpDir is the directory for temporary files
	TmpDir string
	// WorkspaceDir is the directory where projects are stored
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
		TmpDir:       filepath.Join(dataDir, TmpDir),
		WorkspaceDir: filepath.Join(dataDir, ProjectsDir),
	}
}
