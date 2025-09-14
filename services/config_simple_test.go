package services

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetDefaultDataDir(t *testing.T) {
	// This test calls the public function which should return the fixed path
	result := GetDefaultDataDir()

	// Should return the fixed path for native deployment
	expected := "/opt/oar/data"
	assert.Equal(t, expected, result)
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
