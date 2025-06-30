package services

import (
	"testing"
)

func TestGitService_GetLatestCommit_InvalidRepo(t *testing.T) {
	service := NewGitService()

	// Test with non-existent directory
	_, err := service.GetLatestCommit("/non/existent/path")
	if err == nil {
		t.Errorf("GetLatestCommit() expected error for non-existent repository")
	}
}

func TestGitService_Pull_InvalidRepo(t *testing.T) {
	service := NewGitService()

	// Test with non-existent directory
	err := service.Pull("/non/existent/path")
	if err == nil {
		t.Errorf("Pull() expected error for non-existent repository")
	}
}

func TestGitService_Clone_InvalidURL(t *testing.T) {
	service := NewGitService()

	tempDir := t.TempDir()

	// Test with invalid URL
	err := service.Clone("invalid-url", tempDir)
	if err == nil {
		t.Errorf("Clone() expected error for invalid URL")
	}
}

func TestGitService_NewGitService(t *testing.T) {
	service := NewGitService()
	if service == nil {
		t.Errorf("NewGitService() returned nil")
	}
}
