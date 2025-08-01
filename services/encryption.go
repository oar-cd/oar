package services

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fernet/fernet-go"
)

// EncryptionService handles encryption/decryption of sensitive data
type EncryptionService struct {
	key *fernet.Key
}

// NewEncryptionService creates a new encryption service with the provided key
func NewEncryptionService(keyString string) (*EncryptionService, error) {
	if keyString == "" {
		return nil, fmt.Errorf("encryption key cannot be empty")
	}

	key, err := fernet.DecodeKey(keyString)
	if err != nil {
		return nil, fmt.Errorf("invalid encryption key: %w", err)
	}

	return &EncryptionService{key: key}, nil
}

// Encrypt encrypts plaintext and returns a base64-encoded token
func (e *EncryptionService) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil // Don't encrypt empty strings
	}

	token, err := fernet.EncryptAndSign([]byte(plaintext), e.key)
	if err != nil {
		return "", fmt.Errorf("encryption failed: %w", err)
	}
	return base64.StdEncoding.EncodeToString(token), nil
}

// Decrypt decrypts a base64-encoded token and returns plaintext
func (e *EncryptionService) Decrypt(token string) (string, error) {
	if token == "" {
		return "", nil // Return empty string for empty tokens
	}

	// Decode base64 first
	tokenBytes, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return "", fmt.Errorf("invalid token format: %w", err)
	}

	// Set TTL to 100 years - we don't want credentials to expire
	plaintext := fernet.VerifyAndDecrypt(tokenBytes, time.Hour*24*365*100, []*fernet.Key{e.key})
	if plaintext == nil {
		return "", fmt.Errorf("failed to decrypt token: invalid or expired")
	}

	return string(plaintext), nil
}

// GitAuthCredentials represents the structure for storing all Git authentication credentials
type GitAuthCredentials struct {
	HTTP *GitHTTPAuthConfig `json:"http,omitempty"`
	SSH  *GitSSHAuthConfig  `json:"ssh,omitempty"`
}

// EncryptGitAuthConfig encrypts a GitAuthConfig for database storage
func (e *EncryptionService) EncryptGitAuthConfig(
	auth *GitAuthConfig,
) (authType string, encryptedCredentials string, err error) {
	if auth == nil {
		return "", "", nil
	}

	// Convert GitAuthConfig to GitAuthCredentials
	credentials := &GitAuthCredentials{}
	var authTypeValue GitAuthType

	if auth.HTTPAuth != nil {
		credentials.HTTP = auth.HTTPAuth
		authTypeValue = GitAuthTypeHTTP
	}

	if auth.SSHAuth != nil {
		credentials.SSH = auth.SSHAuth
		authTypeValue = GitAuthTypeSSH
	}

	// If no auth is set, return empty
	if credentials.HTTP == nil && credentials.SSH == nil {
		return "", "", nil
	}

	// Serialize credentials
	credentialsData, err := json.Marshal(credentials)
	if err != nil {
		return "", "", fmt.Errorf("failed to serialize credentials: %w", err)
	}

	// Encrypt the serialized data
	encryptedData, err := e.Encrypt(string(credentialsData))
	if err != nil {
		return "", "", fmt.Errorf("failed to encrypt credentials: %w", err)
	}

	return authTypeValue.String(), encryptedData, nil
}

// DecryptGitAuthConfig decrypts credentials back to GitAuthConfig
func (e *EncryptionService) DecryptGitAuthConfig(
	authType string,
	encryptedCredentials string,
) (*GitAuthConfig, error) {
	if authType == "" || encryptedCredentials == "" {
		return nil, nil
	}

	// Parse the authType string into GitAuthType enum
	parsedAuthType, err := ParseGitAuthType(authType)
	if err != nil {
		return nil, fmt.Errorf("invalid auth type: %w", err)
	}

	// Decrypt the serialized data
	credentialsData, err := e.Decrypt(encryptedCredentials)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt credentials: %w", err)
	}

	// Deserialize credentials
	var credentials GitAuthCredentials
	if err := json.Unmarshal([]byte(credentialsData), &credentials); err != nil {
		return nil, fmt.Errorf("failed to deserialize credentials: %w", err)
	}

	// Convert GitAuthCredentials to GitAuthConfig
	auth := &GitAuthConfig{}

	if credentials.HTTP != nil && parsedAuthType == GitAuthTypeHTTP {
		auth.HTTPAuth = credentials.HTTP
	}

	if credentials.SSH != nil && parsedAuthType == GitAuthTypeSSH {
		auth.SSHAuth = credentials.SSH
	}

	return auth, nil
}
