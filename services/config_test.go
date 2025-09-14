package services

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestNewConfigForCLI(t *testing.T) {
	// Use mock environment for testing
	mockEnv := NewMockEnvProvider("/home/testuser", map[string]string{
		"OAR_ENCRYPTION_KEY": generateTestKey(), // Required for config validation
	})
	config, err := NewConfigWithEnv("", mockEnv, WithCLIDefaults())
	if err != nil {
		t.Fatalf("NewConfigForCLI() error = %v", err)
	}

	// Test default behavior
	expectedDataDir := "/opt/oar/data"
	expectedWorkspace := filepath.Join(expectedDataDir, "projects")

	if config.DataDir != expectedDataDir {
		t.Errorf("NewConfigForCLI() DataDir = %v, want %v", config.DataDir, expectedDataDir)
	}
	if config.WorkspaceDir != expectedWorkspace {
		t.Errorf("NewConfigForCLI() WorkspaceDir = %v, want %v", config.WorkspaceDir, expectedWorkspace)
	}
}

func TestNewConfigForWebApp(t *testing.T) {
	// Use mock environment for testing
	mockEnv := NewMockEnvProvider("/home/testuser", map[string]string{
		"OAR_ENCRYPTION_KEY": generateTestKey(), // Required for config validation
	})
	config, err := NewConfigWithEnv("", mockEnv)
	if err != nil {
		t.Fatalf("NewConfigForWebApp() error = %v", err)
	}

	// Test that defaults are set
	expectedDataDir := "/opt/oar/data"
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
	if config.HTTPPort != 3333 {
		t.Errorf("NewConfigForWebApp() HTTPPort = %v, want 3333", config.HTTPPort)
	}
	if config.PollInterval != 5*time.Minute {
		t.Errorf("NewConfigForWebApp() PollInterval = %v, want 5m", config.PollInterval)
	}
}

func TestNewConfigForWebApp_WithEnvVars(t *testing.T) {
	// Use mock environment with custom values
	envVars := map[string]string{
		"OAR_HTTP_PORT":      "3000",
		"OAR_HTTP_HOST":      "0.0.0.0",
		"OAR_POLL_INTERVAL":  "2m",
		"OAR_DATA_DIR":       "/custom/data",
		"OAR_ENCRYPTION_KEY": generateTestKey(), // Required for config validation
	}
	mockEnv := NewMockEnvProvider("/home/testuser", envVars)
	config, err := NewConfigWithEnv("", mockEnv)
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
	if config.DataDir != "/custom/data" {
		t.Errorf("NewConfigForWebApp() DataDir = %v, want /custom/data", config.DataDir)
	}
	if config.PollInterval != 2*time.Minute {
		t.Errorf("NewConfigForWebApp() PollInterval = %v, want 2m", config.PollInterval)
	}
}

func TestConfig_RequiresEncryptionKey(t *testing.T) {
	// Test that config creation fails when encryption key is not provided via environment variable
	mockEnv := NewMockEnvProvider("/home/testuser", map[string]string{
		// Deliberately omit OAR_ENCRYPTION_KEY to test failure
	})

	_, err := NewConfigWithEnv("", mockEnv)
	if err == nil {
		t.Fatal("Expected config creation to fail when encryption key is missing, but got nil error")
	}

	expectedErrorMsg := "encryption key is required - set in config file or OAR_ENCRYPTION_KEY environment variable"
	if !strings.Contains(err.Error(), expectedErrorMsg) {
		t.Errorf("Expected error message to contain %q, got %q", expectedErrorMsg, err.Error())
	}
}

func TestConfig_EncryptionKeyFromEnvironment(t *testing.T) {
	// Test that environment variable is used when set
	envKey := generateTestKey()
	mockEnv := NewMockEnvProvider("/home/testuser", map[string]string{
		"OAR_ENCRYPTION_KEY": envKey,
	})

	config, err := NewConfigWithEnv("", mockEnv)
	if err != nil {
		t.Fatalf("Expected config creation to succeed with env key, but got error: %v", err)
	}

	if config.EncryptionKey != envKey {
		t.Error("Expected encryption key from environment variable to be used")
	}
}
