package services

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetDefaultDataDir(t *testing.T) {
	// This test calls the public function which should work in any environment
	result := GetDefaultDataDir()

	// Should not be empty
	assert.NotEmpty(t, result)

	// Should end with "oar"
	assert.True(t, strings.HasSuffix(result, "oar"))

	// Should contain either .local/share or XDG_DATA_HOME path
	assert.True(t,
		strings.Contains(result, ".local/share") || strings.Contains(result, os.Getenv("XDG_DATA_HOME")),
		"Result should contain .local/share or XDG_DATA_HOME path: %s", result)
}

func TestDefaultEnvProvider_Getenv(t *testing.T) {
	provider := &DefaultEnvProvider{}

	// Test with a known environment variable
	path := provider.Getenv("PATH")
	// PATH should exist in most environments
	assert.NotEmpty(t, path)

	// Test with non-existent variable
	nonExistent := provider.Getenv("DEFINITELY_NON_EXISTENT_VAR_12345")
	assert.Empty(t, nonExistent)
}

func TestDefaultEnvProvider_UserHomeDir(t *testing.T) {
	provider := &DefaultEnvProvider{}

	homeDir, err := provider.UserHomeDir()
	assert.NoError(t, err)
	assert.NotEmpty(t, homeDir)

	// Should be an absolute path
	assert.True(t, strings.HasPrefix(homeDir, "/") || strings.Contains(homeDir, ":"),
		"Home directory should be absolute: %s", homeDir)
}

func TestConfig_GetLogLevel(t *testing.T) {
	tests := []struct {
		name     string
		logLevel string
		expected string
	}{
		{
			name:     "info level",
			logLevel: "info",
			expected: "info",
		},
		{
			name:     "debug level",
			logLevel: "debug",
			expected: "debug",
		},
		{
			name:     "warn level",
			logLevel: "warn",
			expected: "warn",
		},
		{
			name:     "error level",
			logLevel: "error",
			expected: "error",
		},
		{
			name:     "empty level defaults to info",
			logLevel: "",
			expected: "info", // Based on setDefaults()
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				LogLevel: tt.logLevel,
				env:      &DefaultEnvProvider{}, // Need to set env provider
			}
			if tt.logLevel == "" {
				// Test default behavior
				config.setDefaults()
			}

			result := config.GetLogLevel()
			assert.Equal(t, tt.expected, result)
		})
	}
}
