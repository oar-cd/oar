package services

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewConfigForCLI(t *testing.T) {
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
			config, err := NewConfigForCLI(tt.dataDir)
			if err != nil {
				t.Fatalf("NewConfigForCLI() error = %v", err)
			}

			if tt.dataDir == "" {
				// Test default behavior
				homeDir, _ := os.UserHomeDir()
				expectedDataDir := filepath.Join(homeDir, ".oar")
				expectedWorkspace := filepath.Join(expectedDataDir, "projects")

				if config.DataDir != expectedDataDir {
					t.Errorf("NewConfigForCLI() DataDir = %v, want %v", config.DataDir, expectedDataDir)
				}
				if config.WorkspaceDir != expectedWorkspace {
					t.Errorf("NewConfigForCLI() WorkspaceDir = %v, want %v", config.WorkspaceDir, expectedWorkspace)
				}
			} else {
				if config.DataDir != tt.wantDataDir {
					t.Errorf("NewConfigForCLI() DataDir = %v, want %v", config.DataDir, tt.wantDataDir)
				}
				if config.WorkspaceDir != tt.wantWorkspace {
					t.Errorf("NewConfigForCLI() WorkspaceDir = %v, want %v", config.WorkspaceDir, tt.wantWorkspace)
				}
			}
		})
	}
}

func TestNewConfigForWebApp(t *testing.T) {
	config, err := NewConfigForWebApp()
	if err != nil {
		t.Fatalf("NewConfigForWebApp() error = %v", err)
	}

	// Test that defaults are set
	homeDir, _ := os.UserHomeDir()
	expectedDataDir := filepath.Join(homeDir, ".oar")
	expectedWorkspace := filepath.Join(expectedDataDir, "projects")

	if config.DataDir != expectedDataDir {
		t.Errorf("NewConfigForWebApp() DataDir = %v, want %v", config.DataDir, expectedDataDir)
	}
	if config.WorkspaceDir != expectedWorkspace {
		t.Errorf("NewConfigForWebApp() WorkspaceDir = %v, want %v", config.WorkspaceDir, expectedWorkspace)
	}
	if config.HTTPHost != "127.0.0.1" {
		t.Errorf("NewConfigForWebApp() HTTPHost = %v, want 127.0.0.1", config.HTTPHost)
	}
	if config.HTTPPort != 8080 {
		t.Errorf("NewConfigForWebApp() HTTPPort = %v, want 8080", config.HTTPPort)
	}
}
