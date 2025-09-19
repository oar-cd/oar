package services

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfig(t *testing.T) {
	// Create temporary directory and config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-config.yaml")

	// Create minimal valid YAML config
	yamlContent := `
data_dir: /custom/oar/data
log_level: debug
http:
  host: 0.0.0.0
  port: 8080
git:
  timeout: 10m
watcher:
  poll_interval: 2m
encryption_key: nQbG5l9P8YzM2K8vH3FrT1cE4qL7jN6uR0sX9wB2dA8=
`
	err := os.WriteFile(configPath, []byte(yamlContent), 0644)
	require.NoError(t, err)

	// Use mock environment to avoid interference from real env vars
	mockEnv := NewMockEnvProvider("/home/test", map[string]string{})
	config, err := NewConfigWithEnv(configPath, mockEnv)
	assert.NoError(t, err)
	assert.NotNil(t, config)

	// Verify YAML values were loaded
	assert.Equal(t, "/custom/oar/data", config.DataDir)
	assert.Equal(t, "debug", config.LogLevel)
	assert.Equal(t, "0.0.0.0", config.HTTPHost)
	assert.Equal(t, 8080, config.HTTPPort)
	assert.Equal(t, 10*time.Minute, config.GitTimeout)
	assert.Equal(t, 2*time.Minute, config.WatcherPollInterval)
	assert.Equal(t, "nQbG5l9P8YzM2K8vH3FrT1cE4qL7jN6uR0sX9wB2dA8=", config.EncryptionKey)

	// Verify derived paths
	expectedWorkspace := filepath.Join("/custom/oar/data", "projects")
	expectedTmpDir := filepath.Join("/custom/oar/data", "tmp")
	expectedDBPath := filepath.Join("/custom/oar/data", "oar.db")
	assert.Equal(t, expectedWorkspace, config.WorkspaceDir)
	assert.Equal(t, expectedTmpDir, config.TmpDir)
	assert.Equal(t, expectedDBPath, config.DatabasePath)
}

func TestNewConfig_NonExistent(t *testing.T) {
	config, err := NewConfig("/non/existent/config.yaml")
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "failed to load config file")
}

func TestNewConfigWithEnv_NonExistentExplicitPath(t *testing.T) {
	// Test that explicitly provided non-existent config path fails
	mockEnv := NewMockEnvProvider("/home/test", map[string]string{
		"OAR_ENCRYPTION_KEY": generateTestKey(),
	})

	// Non-empty path to non-existent file should fail
	config, err := NewConfigWithEnv("/explicit/non/existent/config.yaml", mockEnv)
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "failed to load config file")
	assert.Contains(t, err.Error(), "failed to read config file /explicit/non/existent/config.yaml")
}

func TestNewConfigWithEnv_EmptyConfigPath(t *testing.T) {
	// Test that empty config path skips YAML loading and uses defaults + env vars
	mockEnv := NewMockEnvProvider("/home/test", map[string]string{
		"OAR_DATA_DIR":       "/env/data/dir",
		"OAR_LOG_LEVEL":      "debug",
		"OAR_HTTP_PORT":      "8080",
		"OAR_ENCRYPTION_KEY": generateTestKey(),
	})

	// Empty config path should skip YAML loading
	config, err := NewConfigWithEnv("", mockEnv)
	assert.NoError(t, err)
	assert.NotNil(t, config)

	// Should use environment variables and defaults (no YAML loaded)
	assert.Equal(t, "/env/data/dir", config.DataDir)
	assert.Equal(t, "debug", config.LogLevel)
	assert.Equal(t, 8080, config.HTTPPort)
	assert.Equal(t, "127.0.0.1", config.HTTPHost) // Default (no env override)

	// Derived paths should be based on env DATA_DIR
	expectedWorkspace := filepath.Join("/env/data/dir", "projects")
	expectedTmpDir := filepath.Join("/env/data/dir", "tmp")
	expectedDBPath := filepath.Join("/env/data/dir", "oar.db")
	assert.Equal(t, expectedWorkspace, config.WorkspaceDir)
	assert.Equal(t, expectedTmpDir, config.TmpDir)
	assert.Equal(t, expectedDBPath, config.DatabasePath)
}

func TestNewConfig_InvalidYAML(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "invalid.yaml")

	// Create invalid YAML
	invalidYAML := `
data_dir: /test
invalid_yaml: [
  - missing closing bracket
encryption_key: test-key
`
	err := os.WriteFile(configPath, []byte(invalidYAML), 0644)
	require.NoError(t, err)

	config, err := NewConfig(configPath)
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "failed to parse YAML config")
}

func TestNewConfigWithEnv_EnvOverrides(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-config.yaml")

	// Create YAML config
	yamlContent := `
data_dir: /yaml/data
log_level: info
http:
  host: 127.0.0.1
  port: 4777
encryption_key: nQbG5l9P8YzM2K8vH3FrT1cE4qL7jN6uR0sX9wB2dA8=
`
	err := os.WriteFile(configPath, []byte(yamlContent), 0644)
	require.NoError(t, err)

	// Create mock environment that overrides some YAML values
	mockEnv := NewMockEnvProvider("/home/test", map[string]string{
		"OAR_DATA_DIR":  "/env/override/data", // Override YAML value
		"OAR_LOG_LEVEL": "debug",              // Override YAML value
		"OAR_HTTP_PORT": "9000",               // Override YAML value
		// Don't override host - should use YAML value
	})

	config, err := NewConfigWithEnv(configPath, mockEnv)
	assert.NoError(t, err)
	assert.NotNil(t, config)

	// Environment variables should override YAML values
	assert.Equal(t, "/env/override/data", config.DataDir)
	assert.Equal(t, "debug", config.LogLevel)
	assert.Equal(t, 9000, config.HTTPPort)

	// Values not overridden by env should come from YAML
	assert.Equal(t, "127.0.0.1", config.HTTPHost)
	assert.Equal(t, "nQbG5l9P8YzM2K8vH3FrT1cE4qL7jN6uR0sX9wB2dA8=", config.EncryptionKey)
}

func TestLoadFromYamlFile_PartialConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "partial-config.yaml")

	// Create minimal YAML config (only some fields)
	yamlContent := `
log_level: warning
http:
  port: 4444
git:
  timeout: 15m
encryption_key: nQbG5l9P8YzM2K8vH3FrT1cE4qL7jN6uR0sX9wB2dA8=
`
	err := os.WriteFile(configPath, []byte(yamlContent), 0644)
	require.NoError(t, err)

	// Use mock environment to avoid interference from real env vars
	mockEnv := NewMockEnvProvider("/home/test", map[string]string{})
	config, err := NewConfigWithEnv(configPath, mockEnv)
	assert.NoError(t, err)
	assert.NotNil(t, config)

	// Specified values should be loaded
	assert.Equal(t, "warning", config.LogLevel)
	assert.Equal(t, 4444, config.HTTPPort)
	assert.Equal(t, 15*time.Minute, config.GitTimeout)

	// Unspecified values should use defaults
	assert.Equal(t, "/opt/oar/data", config.DataDir)
	assert.Equal(t, "127.0.0.1", config.HTTPHost)
	assert.Equal(t, 5*time.Minute, config.WatcherPollInterval)
}

func TestLoadFromYamlFile_InvalidDurations(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "invalid-durations.yaml")

	// Create YAML with invalid duration formats
	yamlContent := `
git:
  timeout: "not-a-duration"
watcher:
  poll_interval: "also-invalid"
encryption_key: nQbG5l9P8YzM2K8vH3FrT1cE4qL7jN6uR0sX9wB2dA8=
`
	err := os.WriteFile(configPath, []byte(yamlContent), 0644)
	require.NoError(t, err)

	// Use mock environment to avoid interference from real env vars
	mockEnv := NewMockEnvProvider("/home/test", map[string]string{})
	config, err := NewConfigWithEnv(configPath, mockEnv)
	assert.NoError(t, err) // Should not error - invalid durations are ignored
	assert.NotNil(t, config)

	// Invalid durations should be ignored, defaults used
	assert.Equal(t, 5*time.Minute, config.GitTimeout)
	assert.Equal(t, 5*time.Minute, config.WatcherPollInterval)
}

func TestNewConfigForCLI_SilentDefault(t *testing.T) {
	mockEnv := NewMockEnvProvider("/home/test", map[string]string{
		"OAR_ENCRYPTION_KEY": generateTestKey(),
	})

	config, err := NewConfigWithEnv("", mockEnv, WithCLIDefaults())
	assert.NoError(t, err)
	assert.NotNil(t, config)

	// CLI should default to silent logging
	assert.Equal(t, "silent", config.LogLevel)
}

func TestNewConfigForWebApp_InfoDefault(t *testing.T) {
	mockEnv := NewMockEnvProvider("/home/test", map[string]string{
		"OAR_ENCRYPTION_KEY": generateTestKey(),
	})

	config, err := NewConfigWithEnv("", mockEnv)
	assert.NoError(t, err)
	assert.NotNil(t, config)

	// WebApp should default to info logging
	assert.Equal(t, "info", config.LogLevel)
}

func TestDerivePaths(t *testing.T) {
	mockEnv := NewMockEnvProvider("/home/test", map[string]string{
		"OAR_DATA_DIR":       "/custom/data/dir",
		"OAR_ENCRYPTION_KEY": generateTestKey(),
	})

	config, err := NewConfigWithEnv("", mockEnv)
	assert.NoError(t, err)
	assert.NotNil(t, config)

	// Test path derivation
	expectedWorkspace := filepath.Join("/custom/data/dir", "projects")
	expectedTmpDir := filepath.Join("/custom/data/dir", "tmp")
	expectedDBPath := filepath.Join("/custom/data/dir", "oar.db")

	assert.Equal(t, expectedWorkspace, config.WorkspaceDir)
	assert.Equal(t, expectedTmpDir, config.TmpDir)
	assert.Equal(t, expectedDBPath, config.DatabasePath)
}

func TestDerivePaths_CustomDatabasePath(t *testing.T) {
	mockEnv := NewMockEnvProvider("/home/test", map[string]string{
		"OAR_DATA_DIR":       "/data",
		"OAR_DATABASE_PATH":  "/custom/db/path.db",
		"OAR_ENCRYPTION_KEY": generateTestKey(),
	})

	config, err := NewConfigWithEnv("", mockEnv)
	assert.NoError(t, err)
	assert.NotNil(t, config)

	// Custom database path should not be overridden
	assert.Equal(t, "/custom/db/path.db", config.DatabasePath)

	// Other paths should still be derived from DataDir
	expectedWorkspace := filepath.Join("/data", "projects")
	expectedTmpDir := filepath.Join("/data", "tmp")
	assert.Equal(t, expectedWorkspace, config.WorkspaceDir)
	assert.Equal(t, expectedTmpDir, config.TmpDir)
}

func TestConfigValidation_LogLevel(t *testing.T) {
	tests := []struct {
		name     string
		logLevel string
		valid    bool
	}{
		{"debug", "debug", true},
		{"info", "info", true},
		{"warning", "warning", true},
		{"error", "error", true},
		{"silent", "silent", true},
		{"invalid", "invalid", false},
		{"warn", "warn", false}, // Should be "warning"
		{"empty", "", true},     // Empty should default to "info"
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEnv := NewMockEnvProvider("/home/test", map[string]string{
				"OAR_LOG_LEVEL":      tt.logLevel,
				"OAR_ENCRYPTION_KEY": generateTestKey(),
			})

			config, err := NewConfigWithEnv("", mockEnv)
			if tt.valid {
				assert.NoError(t, err, "Expected log level '%s' to be valid", tt.logLevel)
				assert.NotNil(t, config)
				if tt.logLevel == "" {
					// Empty log level should default to "info"
					assert.Equal(t, "info", config.LogLevel)
				} else {
					assert.Equal(t, tt.logLevel, config.LogLevel)
				}
			} else {
				assert.Error(t, err, "Expected log level '%s' to be invalid", tt.logLevel)
				assert.Nil(t, config)
				assert.Contains(t, err.Error(), "invalid log level")
			}
		})
	}
}

func TestConfigValidation_HTTPPort(t *testing.T) {
	tests := []struct {
		name  string
		port  string
		valid bool
	}{
		{"port_1", "1", true},
		{"port_4777", "4777", true},
		{"port_8080", "8080", true},
		{"port_65535", "65535", true},
		{"port_0", "0", false},
		{"port_negative", "-1", false},
		{"port_too_high", "65536", false},
		{"port_invalid", "not-a-number", true}, // Invalid strings are ignored, default used
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEnv := NewMockEnvProvider("/home/test", map[string]string{
				"OAR_HTTP_PORT":      tt.port,
				"OAR_ENCRYPTION_KEY": generateTestKey(),
			})

			config, err := NewConfigWithEnv("", mockEnv)
			if tt.valid {
				assert.NoError(t, err, "Expected port '%s' to be valid", tt.port)
				assert.NotNil(t, config)
			} else {
				assert.Error(t, err, "Expected port '%s' to be invalid", tt.port)
				assert.Nil(t, config)
				assert.Contains(t, err.Error(), "invalid HTTP port")
			}
		})
	}
}

func TestConfigValidation_GitTimeout(t *testing.T) {
	tests := []struct {
		name    string
		timeout string
		valid   bool
	}{
		{"valid_1m", "1m", true},
		{"valid_5m", "5m", true},
		{"valid_1h", "1h", true},
		{"valid_30s", "30s", true},
		{"zero_duration", "0s", false},
		{"negative_duration", "-1m", false},
		{"invalid_duration", "invalid", true}, // Invalid durations are ignored, default used
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEnv := NewMockEnvProvider("/home/test", map[string]string{
				"OAR_GIT_TIMEOUT":    tt.timeout,
				"OAR_ENCRYPTION_KEY": generateTestKey(),
			})

			config, err := NewConfigWithEnv("", mockEnv)
			if tt.valid {
				assert.NoError(t, err, "Expected timeout '%s' to be valid", tt.timeout)
				assert.NotNil(t, config)
			} else {
				assert.Error(t, err, "Expected timeout '%s' to be invalid", tt.timeout)
				assert.Nil(t, config)
				assert.Contains(t, err.Error(), "git timeout must be positive")
			}
		})
	}
}

func TestConfigValidation_WatcherPollInterval(t *testing.T) {
	tests := []struct {
		name     string
		interval string
		valid    bool
	}{
		{"valid_1m", "1m", true},
		{"valid_30s", "30s", true},
		{"valid_1h", "1h", true},
		{"zero_interval", "0s", false},
		{"negative_interval", "-1m", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEnv := NewMockEnvProvider("/home/test", map[string]string{
				"OAR_WATCHER_POLL_INTERVAL": tt.interval,
				"OAR_ENCRYPTION_KEY":        generateTestKey(),
			})

			config, err := NewConfigWithEnv("", mockEnv)
			if tt.valid {
				assert.NoError(t, err, "Expected interval '%s' to be valid", tt.interval)
				assert.NotNil(t, config)
			} else {
				assert.Error(t, err, "Expected interval '%s' to be invalid", tt.interval)
				assert.Nil(t, config)
				assert.Contains(t, err.Error(), "watcher poll interval must be positive")
			}
		})
	}
}

// TestConfigValidation_DockerCommand is no longer needed since docker command is hard-coded

func TestConfigValidation_EncryptionKeyRequired(t *testing.T) {
	mockEnv := NewMockEnvProvider("/home/test", map[string]string{
		// Deliberately omit OAR_ENCRYPTION_KEY
	})

	config, err := NewConfigWithEnv("", mockEnv)
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "encryption key is required")
}

func TestYamlConfig_EmptyNestedStructs(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "empty-nested.yaml")

	// Test YAML with empty nested structures
	yamlContent := `
data_dir: /test/data
http:
git:
watcher:
encryption_key: nQbG5l9P8YzM2K8vH3FrT1cE4qL7jN6uR0sX9wB2dA8=
`
	err := os.WriteFile(configPath, []byte(yamlContent), 0644)
	require.NoError(t, err)

	// Use mock environment to avoid interference from real env vars
	mockEnv := NewMockEnvProvider("/home/test", map[string]string{})
	config, err := NewConfigWithEnv(configPath, mockEnv)
	assert.NoError(t, err)
	assert.NotNil(t, config)

	// Should use defaults for empty nested structures
	assert.Equal(t, "127.0.0.1", config.HTTPHost) // Default
	assert.Equal(t, 4777, config.HTTPPort)        // Default
	assert.Equal(t, 5*time.Minute, config.GitTimeout)
	assert.Equal(t, 5*time.Minute, config.WatcherPollInterval)
}
