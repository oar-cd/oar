package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadLines(t *testing.T) {
	tests := []struct {
		name        string
		fileContent string
		expected    []string
		wantErr     bool
	}{
		{
			name:        "single line",
			fileContent: "hello world",
			expected:    []string{"hello world"},
			wantErr:     false,
		},
		{
			name:        "multiple lines",
			fileContent: "line1\nline2\nline3",
			expected:    []string{"line1", "line2", "line3"},
			wantErr:     false,
		},
		{
			name:        "empty file",
			fileContent: "",
			expected:    nil,
			wantErr:     false,
		},
		{
			name:        "file with empty lines",
			fileContent: "line1\n\nline3\n",
			expected:    []string{"line1", "", "line3"},
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.txt")

			err := os.WriteFile(tmpFile, []byte(tt.fileContent), 0644)
			require.NoError(t, err)

			// Test readLines function
			result, err := readLines(tmpFile)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestReadLines_NonExistentFile(t *testing.T) {
	result, err := readLines("/non/existent/file.txt")
	assert.Error(t, err)
	assert.Nil(t, result)
}
