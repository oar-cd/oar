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
	err := service.Pull("/non/existent/path")
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
	err := service.Clone("invalid-url", tempDir)
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
