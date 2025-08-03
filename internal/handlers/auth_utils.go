package handlers

import (
	"net/http"

	"github.com/ch00k/oar/services"
)

// CreateTempAuthConfig creates a temporary GitAuthConfig from form data
// This is a shared utility for handlers that need to process Git authentication
func CreateTempAuthConfig(r *http.Request) *services.GitAuthConfig {
	authType := r.FormValue("auth_type")

	switch authType {
	case "http":
		username := r.FormValue("username")
		password := r.FormValue("password")
		if username != "" && password != "" {
			return &services.GitAuthConfig{
				HTTPAuth: &services.GitHTTPAuthConfig{
					Username: username,
					Password: password,
				},
			}
		}
	case "ssh":
		sshUser := r.FormValue("ssh_user")
		sshPrivateKey := r.FormValue("ssh_private_key")
		if sshPrivateKey != "" {
			// Default SSH user to "git" if not provided
			if sshUser == "" {
				sshUser = "git"
			}
			return &services.GitAuthConfig{
				SSHAuth: &services.GitSSHAuthConfig{
					User:       sshUser,
					PrivateKey: sshPrivateKey,
				},
			}
		}
	}

	// Return nil for "none" auth type or invalid/incomplete auth data
	return nil
}
