package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProjectGitAuthenticationIntegration tests the full flow of project authentication
func TestProjectGitAuthenticationIntegration(t *testing.T) {
	// Setup
	db := setupTestDB(t)
	encryption := setupTestEncryption(t)
	repo := NewProjectRepository(db, encryption)

	// Create a project with HTTP authentication
	originalProject := createTestProject()
	originalProject.GitAuth = &GitAuthConfig{
		HTTPAuth: &GitHTTPAuthConfig{
			Username: "token",
			Password: "ghp_test_secret_token",
		},
	}

	// Save to repository (this should encrypt and store credentials)
	savedProject, err := repo.Create(originalProject)
	require.NoError(t, err)
	require.NotNil(t, savedProject)
	require.NotNil(t, savedProject.GitAuth)
	require.NotNil(t, savedProject.GitAuth.HTTPAuth)

	// Verify credentials are preserved
	assert.Equal(t, "token", savedProject.GitAuth.HTTPAuth.Username)
	assert.Equal(t, "ghp_test_secret_token", savedProject.GitAuth.HTTPAuth.Password)

	// Retrieve from repository (this should decrypt credentials)
	retrievedProject, err := repo.FindByID(savedProject.ID)
	require.NoError(t, err)
	require.NotNil(t, retrievedProject)
	require.NotNil(t, retrievedProject.GitAuth)
	require.NotNil(t, retrievedProject.GitAuth.HTTPAuth)

	// Verify credentials are correctly decrypted
	assert.Equal(t, "token", retrievedProject.GitAuth.HTTPAuth.Username)
	assert.Equal(t, "ghp_test_secret_token", retrievedProject.GitAuth.HTTPAuth.Password)
	assert.Nil(t, retrievedProject.GitAuth.SSHAuth)
}

// TestProjectGitSSHAuthenticationIntegration tests SSH auth integration
func TestProjectGitSSHAuthenticationIntegration(t *testing.T) {
	// Setup
	db := setupTestDB(t)
	encryption := setupTestEncryption(t)
	repo := NewProjectRepository(db, encryption)

	sshKey := loadTestSSHKey(t)

	// Create a project with SSH authentication
	originalProject := createTestProject()
	originalProject.Name = "ssh-test-project"
	originalProject.GitAuth = &GitAuthConfig{
		SSHAuth: &GitSSHAuthConfig{
			PrivateKey: sshKey,
			User:       "git",
		},
	}

	// Save to repository
	savedProject, err := repo.Create(originalProject)
	require.NoError(t, err)
	require.NotNil(t, savedProject.GitAuth)
	require.NotNil(t, savedProject.GitAuth.SSHAuth)

	// Verify credentials are preserved
	assert.Equal(t, sshKey, savedProject.GitAuth.SSHAuth.PrivateKey)
	assert.Equal(t, "git", savedProject.GitAuth.SSHAuth.User)

	// Retrieve from repository
	retrievedProject, err := repo.FindByID(savedProject.ID)
	require.NoError(t, err)
	require.NotNil(t, retrievedProject.GitAuth)
	require.NotNil(t, retrievedProject.GitAuth.SSHAuth)

	// Verify credentials are correctly decrypted
	assert.Equal(t, sshKey, retrievedProject.GitAuth.SSHAuth.PrivateKey)
	assert.Equal(t, "git", retrievedProject.GitAuth.SSHAuth.User)
	assert.Nil(t, retrievedProject.GitAuth.HTTPAuth)
}

// TestProjectWithoutGitAuthentication ensures projects without auth still work
func TestProjectWithoutGitAuthentication(t *testing.T) {
	// Setup
	db := setupTestDB(t)
	encryption := setupTestEncryption(t)
	repo := NewProjectRepository(db, encryption)

	// Create a project without authentication (public repo)
	originalProject := createTestProject()
	originalProject.Name = "public-project"
	originalProject.GitAuth = nil

	// Save to repository
	savedProject, err := repo.Create(originalProject)
	require.NoError(t, err)
	assert.Nil(t, savedProject.GitAuth)

	// Retrieve from repository
	retrievedProject, err := repo.FindByID(savedProject.ID)
	require.NoError(t, err)
	assert.Nil(t, retrievedProject.GitAuth)
}
