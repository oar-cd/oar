package services

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateTestKey(t *testing.T) {
	key1 := generateTestKey()
	key2 := generateTestKey()

	// Keys should be different
	assert.NotEqual(t, key1, key2)

	// Keys should be valid base64
	assert.True(t, len(key1) > 0)
	assert.True(t, len(key2) > 0)

	// Should be able to create encryption service with generated keys
	_, err1 := NewEncryptionService(key1)
	_, err2 := NewEncryptionService(key2)
	assert.NoError(t, err1)
	assert.NoError(t, err2)
}

func TestNewEncryptionService(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{
			name:    "valid key",
			key:     generateTestKey(),
			wantErr: false,
		},
		{
			name:    "empty key",
			key:     "",
			wantErr: true,
		},
		{
			name:    "invalid key",
			key:     "invalid-key",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewEncryptionService(tt.key)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, service)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, service)
			}
		})
	}
}

func TestEncryptionService_EncryptDecrypt(t *testing.T) {
	service, err := NewEncryptionService(generateTestKey())
	require.NoError(t, err)

	tests := []struct {
		name      string
		plaintext string
	}{
		{
			name:      "simple string",
			plaintext: "hello world",
		},
		{
			name:      "empty string",
			plaintext: "",
		},
		{
			name:      "github token",
			plaintext: "ghp_1234567890abcdef",
		},
		{
			name:      "ssh key",
			plaintext: "-----BEGIN OPENSSH PRIVATE KEY-----\nkey content here\n-----END OPENSSH PRIVATE KEY-----",
		},
		{
			name:      "unicode",
			plaintext: "Hello 世界 🚀",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encrypt
			encrypted, err := service.Encrypt(tt.plaintext)
			assert.NoError(t, err)

			if tt.plaintext == "" {
				assert.Equal(t, "", encrypted)
			} else {
				assert.NotEqual(t, tt.plaintext, encrypted)
				assert.True(t, len(encrypted) > 0)
			}

			// Decrypt
			decrypted, err := service.Decrypt(encrypted)
			assert.NoError(t, err)
			assert.Equal(t, tt.plaintext, decrypted)
		})
	}
}

func TestEncryptionService_DecryptInvalidToken(t *testing.T) {
	service, err := NewEncryptionService(generateTestKey())
	require.NoError(t, err)

	// Try to decrypt invalid tokens
	invalidTokens := []string{
		"invalid-token",
		"gAAAAABh",
		"completely-wrong",
	}

	for _, token := range invalidTokens {
		_, err := service.Decrypt(token)
		assert.Error(t, err)
		// Error could be either "invalid token format" or "failed to decrypt token"
		assert.True(t,
			strings.Contains(err.Error(), "failed to decrypt token") ||
				strings.Contains(err.Error(), "invalid token format"))
	}
}

func TestEncryptionService_EncryptDecryptGitSSHAuthDefaultUser(t *testing.T) {
	service, err := NewEncryptionService(generateTestKey())
	require.NoError(t, err)

	sshKey := loadTestSSHKey(t)
	auth := &GitAuthConfig{
		SSHAuth: &GitSSHAuthConfig{
			PrivateKey: sshKey,
			User:       "", // EncryptGitAuthConfig method preserves empty value (no default)
		},
	}

	// Encrypt using EncryptGitAuthConfig method
	authType, encryptedCredentials, err := service.EncryptGitAuthConfig(auth)
	require.NoError(t, err)

	// Decrypt using DecryptGitAuthConfig method
	decrypted, err := service.DecryptGitAuthConfig(authType, encryptedCredentials)
	require.NoError(t, err)
	require.NotNil(t, decrypted.SSHAuth)

	// EncryptGitAuthConfig method preserves the original empty value (no automatic default to "git")
	assert.Equal(t, "", decrypted.SSHAuth.User)
}

func TestEncryptionService_EncryptDecryptGitAuthConfig_HTTP(t *testing.T) {
	service, err := NewEncryptionService(generateTestKey())
	require.NoError(t, err)

	auth := &GitAuthConfig{
		HTTPAuth: &GitHTTPAuthConfig{
			Username: "token",
			Password: "ghp_1234567890abcdef",
		},
	}

	// Encrypt
	authType, encryptedCredentials, err := service.EncryptGitAuthConfig(auth)
	require.NoError(t, err)

	assert.Equal(t, "http", authType)
	assert.NotEmpty(t, encryptedCredentials)
	assert.NotContains(t, encryptedCredentials, "token")
	assert.NotContains(t, encryptedCredentials, "ghp_1234567890abcdef")

	// Decrypt
	decrypted, err := service.DecryptGitAuthConfig(authType, encryptedCredentials)
	require.NoError(t, err)
	require.NotNil(t, decrypted)
	require.NotNil(t, decrypted.HTTPAuth)
	assert.Nil(t, decrypted.SSHAuth)

	assert.Equal(t, "token", decrypted.HTTPAuth.Username)
	assert.Equal(t, "ghp_1234567890abcdef", decrypted.HTTPAuth.Password)
}

func TestEncryptionService_EncryptDecryptGitAuthConfig_SSH(t *testing.T) {
	service, err := NewEncryptionService(generateTestKey())
	require.NoError(t, err)

	sshKey := loadTestSSHKey(t)
	auth := &GitAuthConfig{
		SSHAuth: &GitSSHAuthConfig{
			PrivateKey: sshKey,
			User:       "testuser",
		},
	}

	// Encrypt
	authType, encryptedCredentials, err := service.EncryptGitAuthConfig(auth)
	require.NoError(t, err)

	assert.Equal(t, "ssh", authType)
	assert.NotEmpty(t, encryptedCredentials)
	assert.NotContains(t, encryptedCredentials, "testuser")
	assert.NotContains(t, encryptedCredentials, "BEGIN OPENSSH PRIVATE KEY")

	// Decrypt
	decrypted, err := service.DecryptGitAuthConfig(authType, encryptedCredentials)
	require.NoError(t, err)
	require.NotNil(t, decrypted)
	require.NotNil(t, decrypted.SSHAuth)
	assert.Nil(t, decrypted.HTTPAuth)

	assert.Equal(t, sshKey, decrypted.SSHAuth.PrivateKey)
	assert.Equal(t, "testuser", decrypted.SSHAuth.User)
}

func TestEncryptionService_EncryptDecryptGitAuthConfig_Nil(t *testing.T) {
	service, err := NewEncryptionService(generateTestKey())
	require.NoError(t, err)

	// Encrypt nil
	authType, encryptedCredentials, err := service.EncryptGitAuthConfig(nil)
	assert.NoError(t, err)
	assert.Empty(t, authType)
	assert.Empty(t, encryptedCredentials)

	// Decrypt nil
	decrypted, err := service.DecryptGitAuthConfig("", "")
	assert.NoError(t, err)
	assert.Nil(t, decrypted)
}

func TestEncryptionService_EncryptDecryptGitAuthConfig_Empty(t *testing.T) {
	service, err := NewEncryptionService(generateTestKey())
	require.NoError(t, err)

	auth := &GitAuthConfig{} // Empty config

	authType, encryptedCredentials, err := service.EncryptGitAuthConfig(auth)
	assert.NoError(t, err)
	assert.Empty(t, authType)
	assert.Empty(t, encryptedCredentials)
}

func TestEncryptionService_DecryptionErrors(t *testing.T) {
	service, err := NewEncryptionService(generateTestKey())
	require.NoError(t, err)

	tests := []struct {
		name                 string
		authType             string
		encryptedCredentials string
		wantErr              string
	}{
		{
			name:                 "invalid auth type",
			authType:             "unknown",
			encryptedCredentials: "some-encrypted-data",
			wantErr:              "invalid auth type",
		},
		{
			name:                 "invalid encrypted data",
			authType:             "http",
			encryptedCredentials: "invalid-base64-data",
			wantErr:              "failed to decrypt",
		},
		{
			name:                 "empty auth type with data",
			authType:             "",
			encryptedCredentials: "some-data",
			wantErr:              "", // Should return nil, nil
		},
		{
			name:     "valid encryption but invalid data format",
			authType: "http",
			encryptedCredentials: func() string {
				encrypted, _ := service.Encrypt("invalid data {")
				return encrypted
			}(),
			wantErr: "failed to deserialize",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.DecryptGitAuthConfig(tt.authType, tt.encryptedCredentials)
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}
