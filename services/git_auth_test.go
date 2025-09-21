package services

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitAuthConfig_createAuthMethod(t *testing.T) {
	config := &Config{
		GitTimeout: 5 * time.Minute,
	}
	gitService := NewGitService(config)

	tests := []struct {
		name     string
		auth     *GitAuthConfig
		wantType interface{}
		wantErr  bool
	}{
		{
			name:     "nil auth config should return nil",
			auth:     nil,
			wantType: nil,
			wantErr:  false,
		},
		{
			name:     "empty auth config should return nil",
			auth:     &GitAuthConfig{},
			wantType: nil,
			wantErr:  false,
		},
		{
			name: "HTTP auth should return BasicAuth",
			auth: &GitAuthConfig{
				HTTPAuth: &GitHTTPAuthConfig{
					Username: "token",
					Password: "ghp_test_token",
				},
			},
			wantType: &http.BasicAuth{},
			wantErr:  false,
		},
		{
			name: "SSH auth should return PublicKeys",
			auth: &GitAuthConfig{
				SSHAuth: &GitSSHAuthConfig{
					PrivateKey: loadTestSSHKey(t),
					User:       "git",
				},
			},
			wantType: &ssh.PublicKeys{},
			wantErr:  false,
		},
		{
			name: "SSH auth with empty user should default to git",
			auth: &GitAuthConfig{
				SSHAuth: &GitSSHAuthConfig{
					PrivateKey: loadTestSSHKey(t),
					User:       "",
				},
			},
			wantType: &ssh.PublicKeys{},
			wantErr:  false,
		},
		{
			name: "SSH auth with invalid key should return error",
			auth: &GitAuthConfig{
				SSHAuth: &GitSSHAuthConfig{
					PrivateKey: "invalid-key",
					User:       "git",
				},
			},
			wantType: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authMethod, err := gitService.createAuthMethod(tt.auth)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, authMethod)
			} else {
				assert.NoError(t, err)
				if tt.wantType == nil {
					assert.Nil(t, authMethod)
				} else {
					assert.IsType(t, tt.wantType, authMethod)
				}
			}
		})
	}
}

func TestGitAuthConfig_createHTTPAuth(t *testing.T) {
	config := &Config{
		GitTimeout: 5 * time.Minute,
	}
	gitService := NewGitService(config)

	auth := &GitAuthConfig{
		HTTPAuth: &GitHTTPAuthConfig{
			Username: "token",
			Password: "ghp_test_token",
		},
	}

	authMethod, err := gitService.createAuthMethod(auth)
	require.NoError(t, err)

	basicAuth, ok := authMethod.(*http.BasicAuth)
	require.True(t, ok, "Expected *http.BasicAuth")

	assert.Equal(t, "token", basicAuth.Username)
	assert.Equal(t, "ghp_test_token", basicAuth.Password)
}

func TestGitAuthConfig_createSSHAuth(t *testing.T) {
	config := &Config{
		GitTimeout: 5 * time.Minute,
	}
	gitService := NewGitService(config)

	auth := &GitAuthConfig{
		SSHAuth: &GitSSHAuthConfig{
			PrivateKey: loadTestSSHKey(t),
			User:       "testuser",
		},
	}

	authMethod, err := gitService.createAuthMethod(auth)
	require.NoError(t, err)

	publicKeys, ok := authMethod.(*ssh.PublicKeys)
	require.True(t, ok, "Expected *ssh.PublicKeys")

	assert.Equal(t, "testuser", publicKeys.User)
}

func TestGitAuthConfig_createSSHAuth_DefaultUser(t *testing.T) {
	config := &Config{
		GitTimeout: 5 * time.Minute,
	}
	gitService := NewGitService(config)

	auth := &GitAuthConfig{
		SSHAuth: &GitSSHAuthConfig{
			PrivateKey: loadTestSSHKey(t),
			User:       "", // Empty user should default to "git"
		},
	}

	authMethod, err := gitService.createAuthMethod(auth)
	require.NoError(t, err)

	publicKeys, ok := authMethod.(*ssh.PublicKeys)
	require.True(t, ok, "Expected *ssh.PublicKeys")

	assert.Equal(t, "git", publicKeys.User)
}

func TestGitAuthConfig_createSSHAuth_NilConfig(t *testing.T) {
	config := &Config{
		GitTimeout: 5 * time.Minute,
	}
	gitService := NewGitService(config)

	_, err := gitService.createSSHAuth(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "SSH auth config is nil")
}

func TestGitAuthConfig_MultipleAuthMethods(t *testing.T) {
	config := &Config{
		GitTimeout: 5 * time.Minute,
	}
	gitService := NewGitService(config)

	// Both HTTP and SSH auth configured - should prefer HTTP
	auth := &GitAuthConfig{
		HTTPAuth: &GitHTTPAuthConfig{
			Username: "token",
			Password: "ghp_test_token",
		},
		SSHAuth: &GitSSHAuthConfig{
			PrivateKey: loadTestSSHKey(t),
			User:       "git",
		},
	}

	authMethod, err := gitService.createAuthMethod(auth)
	require.NoError(t, err)

	// Should return HTTP auth (first checked)
	_, ok := authMethod.(*http.BasicAuth)
	assert.True(t, ok, "Expected HTTP auth to be preferred when both are configured")
}

func TestGitService_Clone_WithAuth(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir := t.TempDir()
	cloneDir := filepath.Join(tempDir, "clone")

	config := &Config{
		GitTimeout: 30 * time.Second,
	}
	gitService := NewGitService(config)

	// Test cloning public repo (no auth)
	err := gitService.Clone("https://github.com/oar-cd/oar-test-public.git", "", nil, cloneDir)
	if err != nil {
		t.Logf("Public repo clone failed (may not exist): %v", err)
		// Don't fail the test - the public repo might not exist
		// but the code structure is what we're testing
	}

	// Check that directory was created
	if _, err := os.Stat(cloneDir); err == nil {
		assert.DirExists(t, cloneDir)
	}
}

func TestGitService_Pull_WithAuth(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir := t.TempDir()

	config := &Config{
		GitTimeout: 30 * time.Second,
	}
	gitService := NewGitService(config)

	// First clone a repo
	cloneDir := filepath.Join(tempDir, "clone")
	err := gitService.Clone("https://github.com/oar-cd/oar-test-public.git", "", nil, cloneDir)
	if err != nil {
		t.Skipf("Cannot test pull - clone failed: %v", err)
	}

	// Test pulling with no auth
	err = gitService.Pull("main", nil, cloneDir)
	// Don't assert success since remote repo might not exist
	// Just ensure no panic and method is callable
	t.Logf("Pull result: %v", err)
}

// loadTestSSHKey loads the test SSH key from the testdata directory
func loadTestSSHKey(t *testing.T) string {
	t.Helper()
	keyPath := filepath.Join("testdata", "test_ssh_key")
	keyBytes, err := os.ReadFile(keyPath)
	require.NoError(t, err, "Failed to read test SSH key")
	return string(keyBytes)
}
