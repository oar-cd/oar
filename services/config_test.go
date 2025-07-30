package services

import (
	"path/filepath"
	"testing"
)

// MockEnvProvider implements EnvProvider for testing
type MockEnvProvider struct {
	envVars map[string]string
	homeDir string
}

func NewMockEnvProvider(homeDir string, envVars map[string]string) *MockEnvProvider {
	if envVars == nil {
		envVars = make(map[string]string)
	}
	return &MockEnvProvider{
		envVars: envVars,
		homeDir: homeDir,
	}
}

func (m *MockEnvProvider) Getenv(key string) string {
	return m.envVars[key]
}

func (m *MockEnvProvider) UserHomeDir() (string, error) {
	return m.homeDir, nil
}

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
			// Use mock environment for testing
			mockEnv := NewMockEnvProvider("/home/testuser", nil)
			config, err := NewConfigForCLIWithEnv(mockEnv, tt.dataDir)
			if err != nil {
				t.Fatalf("NewConfigForCLI() error = %v", err)
			}

			if tt.dataDir == "" {
				// Test default behavior with mock environment
				expectedDataDir := getDefaultDataDirWithEnv(mockEnv)
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
	// Use mock environment for testing (no environment variables set)
	mockEnv := NewMockEnvProvider("/home/testuser", nil)
	config, err := NewConfigForWebAppWithEnv(mockEnv)
	if err != nil {
		t.Fatalf("NewConfigForWebApp() error = %v", err)
	}

	// Test that defaults are set
	expectedDataDir := getDefaultDataDirWithEnv(mockEnv)
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

func TestNewConfigForWebApp_WithEnvVars(t *testing.T) {
	// Use mock environment with custom values
	envVars := map[string]string{
		"OAR_HTTP_PORT": "3000",
		"OAR_HTTP_HOST": "0.0.0.0",
		"XDG_DATA_HOME": "/custom/data",
	}
	mockEnv := NewMockEnvProvider("/home/testuser", envVars)
	config, err := NewConfigForWebAppWithEnv(mockEnv)
	if err != nil {
		t.Fatalf("NewConfigForWebApp() error = %v", err)
	}

	// Test that environment variables are respected
	if config.HTTPPort != 3000 {
		t.Errorf("NewConfigForWebApp() HTTPPort = %v, want 3000", config.HTTPPort)
	}
	if config.HTTPHost != "0.0.0.0" {
		t.Errorf("NewConfigForWebApp() HTTPHost = %v, want 0.0.0.0", config.HTTPHost)
	}
	if config.DataDir != "/custom/data/oar" {
		t.Errorf("NewConfigForWebApp() DataDir = %v, want /custom/data/oar", config.DataDir)
	}
}
