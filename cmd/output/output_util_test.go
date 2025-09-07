package output

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaskSensitiveValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty value",
			input:    "",
			expected: "(not set)",
		},
		{
			name:     "single character",
			input:    "a",
			expected: "*",
		},
		{
			name:     "two characters",
			input:    "ab",
			expected: "**",
		},
		{
			name:     "three characters",
			input:    "abc",
			expected: "a*c",
		},
		{
			name:     "short value (4-8 chars)",
			input:    "secret",
			expected: "s****t",
		},
		{
			name:     "exactly 8 characters",
			input:    "password",
			expected: "p******d",
		},
		{
			name:     "long value (>8 chars)",
			input:    "verylongsecretpassword",
			expected: "ver****************ord",
		},
		{
			name:     "github token example",
			input:    "ghp_1234567890abcdef",
			expected: "ghp**************def",
		},
		{
			name:     "short token",
			input:    "token123",
			expected: "t******3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskSensitiveValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatCommitDetails(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty commit",
			input:    "",
			expected: "(no commits)",
		},
		{
			name:     "short commit (8 chars or less)",
			input:    "abc123",
			expected: "abc123",
		},
		{
			name:     "exactly 8 characters",
			input:    "12345678",
			expected: "12345678",
		},
		{
			name:     "long commit hash",
			input:    "1234567890abcdef1234567890abcdef12345678",
			expected: "12345678 (1234567890abcdef1234567890abcdef12345678)",
		},
		{
			name:     "typical git hash",
			input:    "a1b2c3d4e5f6789012345678901234567890abcd",
			expected: "a1b2c3d4 (a1b2c3d4e5f6789012345678901234567890abcd)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatCommitDetails(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatCommitHash(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty commit",
			input:    "",
			expected: "-",
		},
		{
			name:     "short commit",
			input:    "abc123",
			expected: "abc123",
		},
		{
			name:     "exactly 8 characters",
			input:    "12345678",
			expected: "12345678",
		},
		{
			name:     "long commit hash",
			input:    "1234567890abcdef1234567890abcdef12345678",
			expected: "12345678",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatCommitHash(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		maxLength int
		expected  string
	}{
		{
			name:      "string shorter than max",
			input:     "hello",
			maxLength: 10,
			expected:  "hello",
		},
		{
			name:      "string equal to max",
			input:     "hello",
			maxLength: 5,
			expected:  "hello",
		},
		{
			name:      "string longer than max",
			input:     "hello world",
			maxLength: 8,
			expected:  "hello...",
		},
		{
			name:      "very short max length",
			input:     "hello world",
			maxLength: 3,
			expected:  "...",
		},
		{
			name:      "max length 4",
			input:     "hello world",
			maxLength: 4,
			expected:  "h...",
		},
		{
			name:      "empty string",
			input:     "",
			maxLength: 5,
			expected:  "",
		},
		{
			name:      "single character with max 1",
			input:     "a",
			maxLength: 1,
			expected:  "a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateString(tt.input, tt.maxLength)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatStringList(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected string
	}{
		{
			name:     "empty list",
			input:    []string{},
			expected: "(none)",
		},
		{
			name:     "nil list",
			input:    nil,
			expected: "(none)",
		},
		{
			name:     "single item",
			input:    []string{"item1"},
			expected: "item1",
		},
		{
			name:     "two items",
			input:    []string{"item1", "item2"},
			expected: "1. item1\n2. item2",
		},
		{
			name:     "multiple items",
			input:    []string{"docker-compose.yml", "docker-compose.prod.yml", "docker-compose.dev.yml"},
			expected: "1. docker-compose.yml\n2. docker-compose.prod.yml\n3. docker-compose.dev.yml",
		},
		{
			name:     "items with spaces",
			input:    []string{"first item", "second item with spaces", "third"},
			expected: "1. first item\n2. second item with spaces\n3. third",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatStringList(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
