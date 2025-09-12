package services

import (
	"testing"
	"time"
)

func TestGitService_GetLatestCommit_InvalidRepo(t *testing.T) {
	config := &Config{
		GitTimeout: 5 * time.Minute,
	}
	service := NewGitService(config)

	// Test with non-existent directory
	_, err := service.GetLatestCommit("/non/existent/path")
	if err == nil {
		t.Errorf("GetLatestCommit() expected error for non-existent repository")
	}
}

func TestGitService_Pull_InvalidRepo(t *testing.T) {
	config := &Config{
		GitTimeout: 5 * time.Minute,
	}
	service := NewGitService(config)

	// Test with non-existent directory
	err := service.Pull("", nil, "/non/existent/path")
	if err == nil {
		t.Errorf("Pull() expected error for non-existent repository")
	}
}

func TestGitService_Clone_InvalidURL(t *testing.T) {
	config := &Config{
		GitTimeout: 5 * time.Minute,
	}
	service := NewGitService(config)

	tempDir := t.TempDir()

	// Test with invalid URL
	err := service.Clone("invalid-url", "", nil, tempDir)
	if err == nil {
		t.Errorf("Clone() expected error for invalid URL")
	}
}

func TestGitService_NewGitService(t *testing.T) {
	config := &Config{
		GitTimeout: 5 * time.Minute,
	}
	service := NewGitService(config)
	if service == nil {
		t.Errorf("NewGitService() returned nil")
	}
}

func TestGitService_TimeoutConfiguration(t *testing.T) {
	// Test that GitService properly stores the timeout from config
	config := &Config{
		GitTimeout: 30 * time.Second,
	}
	service := NewGitService(config)

	if service.config.GitTimeout != 30*time.Second {
		t.Errorf("GitService config timeout = %v, want %v", service.config.GitTimeout, 30*time.Second)
	}
}

func TestGitAuthType_String(t *testing.T) {
	tests := []struct {
		authType GitAuthType
		expected string
	}{
		{GitAuthTypeHTTP, "http"},
		{GitAuthTypeSSH, "ssh"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.authType.String(); got != tt.expected {
				t.Errorf("GitAuthType.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGitAuthType_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		authType GitAuthType
		expected bool
	}{
		{"valid HTTP", GitAuthTypeHTTP, true},
		{"valid SSH", GitAuthTypeSSH, true},
		{"invalid empty", GitAuthType(""), false},
		{"invalid unknown", GitAuthType("unknown"), false},
		{"invalid oauth", GitAuthType("oauth"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.authType.IsValid(); got != tt.expected {
				t.Errorf("GitAuthType.IsValid() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParseGitAuthType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected GitAuthType
		wantErr  bool
	}{
		{"valid HTTP", "http", GitAuthTypeHTTP, false},
		{"valid SSH", "ssh", GitAuthTypeSSH, false},
		{"invalid empty", "", GitAuthType(""), true},
		{"invalid unknown", "unknown", GitAuthType(""), true},
		{"invalid oauth", "oauth", GitAuthType(""), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseGitAuthType(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseGitAuthType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("ParseGitAuthType() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGitService_Clone_WithBranch(t *testing.T) {
	config := &Config{
		GitTimeout: 5 * time.Minute,
	}
	service := NewGitService(config)

	tempDir := t.TempDir()

	// Test with specific branch (this will fail but we're testing the interface)
	err := service.Clone("invalid-url", "main", nil, tempDir)
	if err == nil {
		t.Errorf("Clone() expected error for invalid URL with branch")
	}

	// Test with empty branch (default branch)
	err = service.Clone("invalid-url", "", nil, tempDir)
	if err == nil {
		t.Errorf("Clone() expected error for invalid URL with default branch")
	}
}

func TestGitService_Pull_WithBranch(t *testing.T) {
	config := &Config{
		GitTimeout: 5 * time.Minute,
	}
	service := NewGitService(config)

	// Test with specific branch
	err := service.Pull("main", nil, "/non/existent/path")
	if err == nil {
		t.Errorf("Pull() expected error for non-existent repository with branch")
	}

	// Test with empty branch (default branch)
	err = service.Pull("", nil, "/non/existent/path")
	if err == nil {
		t.Errorf("Pull() expected error for non-existent repository with default branch")
	}
}

func TestGitService_Fetch_InvalidRepo(t *testing.T) {
	config := &Config{
		GitTimeout: 5 * time.Minute,
	}
	service := NewGitService(config)

	// Test with non-existent directory
	err := service.Fetch("main", nil, "/non/existent/path")
	if err == nil {
		t.Errorf("Fetch() expected error for non-existent repository")
	}
}

func TestGitService_Fetch_WithBranch(t *testing.T) {
	config := &Config{
		GitTimeout: 5 * time.Minute,
	}
	service := NewGitService(config)

	// Test with specific branch
	err := service.Fetch("main", nil, "/non/existent/path")
	if err == nil {
		t.Errorf("Fetch() expected error for non-existent repository with branch")
	}

	// Test with empty branch (default branch)
	err = service.Fetch("", nil, "/non/existent/path")
	if err == nil {
		t.Errorf("Fetch() expected error for non-existent repository with default branch")
	}
}

func TestGitService_GetRemoteLatestCommit_InvalidRepo(t *testing.T) {
	config := &Config{
		GitTimeout: 5 * time.Minute,
	}
	service := NewGitService(config)

	// Test with non-existent directory
	_, err := service.GetRemoteLatestCommit("/non/existent/path", "main")
	if err == nil {
		t.Errorf("GetRemoteLatestCommit() expected error for non-existent repository")
	}
}

func TestGitService_GetRemoteLatestCommit_WithBranch(t *testing.T) {
	config := &Config{
		GitTimeout: 5 * time.Minute,
	}
	service := NewGitService(config)

	// Test with specific branch
	_, err := service.GetRemoteLatestCommit("/non/existent/path", "main")
	if err == nil {
		t.Errorf("GetRemoteLatestCommit() expected error for non-existent repository with branch")
	}

	// Test with empty branch (should still fail on non-existent repo)
	_, err = service.GetRemoteLatestCommit("/non/existent/path", "")
	if err == nil {
		t.Errorf("GetRemoteLatestCommit() expected error for non-existent repository with empty branch")
	}
}
