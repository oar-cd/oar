package services

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewConfig(t *testing.T) {
	tests := []struct {
		name          string
		dataDir       string
		wantDataDir   string
		wantWorkspace string
	}{
		{
			name:          "custom data directory",
			dataDir:       "/custom/path",
			wantDataDir:   "/custom/path",
			wantWorkspace: "/custom/path/projects",
		},
		{
			name:          "empty data directory uses default",
			dataDir:       "",
			wantDataDir:   "", // Will be set to home/.oar
			wantWorkspace: "", // Will be set to home/.oar/projects
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := NewConfig(tt.dataDir)

			if tt.dataDir == "" {
				// Test default behavior
				homeDir, _ := os.UserHomeDir()
				expectedDataDir := filepath.Join(homeDir, ".oar")
				expectedWorkspace := filepath.Join(expectedDataDir, "projects")

				if config.DataDir != expectedDataDir {
					t.Errorf("NewConfig() DataDir = %v, want %v", config.DataDir, expectedDataDir)
				}
				if config.WorkspaceDir != expectedWorkspace {
					t.Errorf("NewConfig() WorkspaceDir = %v, want %v", config.WorkspaceDir, expectedWorkspace)
				}
			} else {
				if config.DataDir != tt.wantDataDir {
					t.Errorf("NewConfig() DataDir = %v, want %v", config.DataDir, tt.wantDataDir)
				}
				if config.WorkspaceDir != tt.wantWorkspace {
					t.Errorf("NewConfig() WorkspaceDir = %v, want %v", config.WorkspaceDir, tt.wantWorkspace)
				}
			}
		})
	}
}
