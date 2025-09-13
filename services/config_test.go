package services

import (
	"fmt"
	"os"
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

func (m *MockEnvProvider) UserHomeDir() (string, error) {
	return m.homeDir, nil
}

func TestNewConfigForCLI(t *testing.T) {
	// Use mock environment for testing
	mockEnv := NewMockEnvProvider("/home/testuser", map[string]string{
		"OAR_ENCRYPTION_KEY": generateTestKey(), // Required for config validation
	})
	config, err := NewConfigForCLIWithEnv(mockEnv)
	if err != nil {
		t.Fatalf("NewConfigForCLI() error = %v", err)
	}

	// Test default behavior with mock environment
	expectedInstallDir := getDefaultInstallDirWithEnv(mockEnv)
	expectedDataDir := filepath.Join(expectedInstallDir, "data")
	expectedWorkspace := filepath.Join(expectedDataDir, "projects")

	if config.InstallDir != expectedInstallDir {
		t.Errorf("NewConfigForCLI() InstallDir = %v, want %v", config.InstallDir, expectedInstallDir)
	}
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
	config, err := NewConfigForWebAppWithEnv(mockEnv)
	if err != nil {
		t.Fatalf("NewConfigForWebApp() error = %v", err)
	}

	// Test that defaults are set
	expectedInstallDir := getDefaultInstallDirWithEnv(mockEnv)
	expectedDataDir := filepath.Join(expectedInstallDir, "data")
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
		"XDG_DATA_HOME":      "/custom/data",
		"OAR_ENCRYPTION_KEY": generateTestKey(), // Required for config validation
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
	if config.DataDir != "/custom/data/oar/data" {
		t.Errorf("NewConfigForWebApp() DataDir = %v, want /custom/data/oar/data", config.DataDir)
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

	_, err := NewConfigForWebAppWithEnv(mockEnv)
	if err == nil {
		t.Fatal("Expected config creation to fail when encryption key is missing, but got nil error")
	}

	expectedErrorMsg := "encryption key is required - set OAR_ENCRYPTION_KEY environment variable or ensure .env file exists in data directory"
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

	config, err := NewConfigForWebAppWithEnv(mockEnv)
	if err != nil {
		t.Fatalf("Expected config creation to succeed with env key, but got error: %v", err)
	}

	if config.EncryptionKey != envKey {
		t.Error("Expected encryption key from environment variable to be used")
	}
}

func TestConfig_EncryptionKeyFromEnvFile(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()

	// Create .env file with encryption key
	envKey := generateTestKey()
	envContent := fmt.Sprintf(`# Test .env file
OAR_UID=1000
OAR_GID=1000

# Encryption key
OAR_ENCRYPTION_KEY=%s
`, envKey)

	envFile := filepath.Join(tempDir, ".env")
	if err := os.WriteFile(envFile, []byte(envContent), 0o644); err != nil {
		t.Fatalf("Failed to create test .env file: %v", err)
	}

	// Create config with no environment variable set (fallback to .env file)
	mockEnv := NewMockEnvProvider("/home/testuser", map[string]string{
		"OAR_INSTALL_DIR": tempDir, // Set install dir to temp dir for .env file lookup
	})

	config, err := NewConfigForCLIWithEnv(mockEnv)
	if err != nil {
		t.Fatalf("Expected config creation to succeed with .env file, got error: %v", err)
	}

	if config.EncryptionKey != envKey {
		t.Errorf("Expected encryption key from .env file (%q) to be used, got %q", envKey, config.EncryptionKey)
	}
}

func TestConfig_EncryptionKeyEnvironmentOverridesEnvFile(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()

	// Create .env file with one encryption key
	envFileKey := generateTestKey()
	envContent := fmt.Sprintf("OAR_ENCRYPTION_KEY=%s\n", envFileKey)

	envFile := filepath.Join(tempDir, ".env")
	if err := os.WriteFile(envFile, []byte(envContent), 0o644); err != nil {
		t.Fatalf("Failed to create test .env file: %v", err)
	}

	// Set environment variable with different key (should override .env file)
	envVarKey := generateTestKey()
	mockEnv := NewMockEnvProvider("/home/testuser", map[string]string{
		"OAR_ENCRYPTION_KEY": envVarKey,
		"OAR_INSTALL_DIR":    tempDir, // Set install dir to temp dir for .env file lookup
	})

	config, err := NewConfigForCLIWithEnv(mockEnv)
	if err != nil {
		t.Fatalf("Expected config creation to succeed, got error: %v", err)
	}

	if config.EncryptionKey != envVarKey {
		t.Errorf(
			"Expected environment variable encryption key (%q) to override .env file, got %q",
			envVarKey,
			config.EncryptionKey,
		)
	}
}
