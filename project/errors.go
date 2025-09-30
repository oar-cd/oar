package project

import (
	"strings"
)

// FormatErrorForUser converts technical errors to user-friendly messages
// This should only be called at the handler level
func FormatErrorForUser(err error) string {
	if err == nil {
		return ""
	}

	errStr := strings.ToLower(err.Error())

	switch {
	case strings.Contains(errStr, "unique constraint") && strings.Contains(errStr, "name"):
		return "a project with this name already exists"
	case strings.Contains(errStr, "unique constraint"):
		return "this entry already exists"
	case strings.Contains(errStr, "record not found"):
		return "project not found"
	case strings.Contains(errStr, "connection"):
		return "database connection failed"
	case strings.Contains(errStr, "timeout"):
		return "operation timed out"
	// Git authentication failures (more specific matches first)
	case strings.Contains(errStr, "permission denied (publickey)"):
		return "ssh key authentication failed - please check your private key"
	case strings.Contains(errStr, "host key verification failed"):
		return "ssh host key verification failed - please check your SSH configuration"
	case strings.Contains(errStr, "authentication failed"):
		return "git authentication failed - please check your credentials"
	case strings.Contains(errStr, "could not read username") || strings.Contains(errStr, "could not read password"):
		return "git authentication required - please provide valid credentials"
	case strings.Contains(errStr, "terminal prompts disabled"):
		return "git authentication required - repository needs credentials to access"
	case strings.Contains(errStr, "access denied") && strings.Contains(errStr, "git"):
		return "git access denied - please check your repository permissions"
	case strings.Contains(errStr, "repository not found") || strings.Contains(errStr, "not found"):
		return "git repository not found - please check the URL and your access permissions"
	case strings.Contains(errStr, "invalid credentials"):
		return "invalid git credentials - please check your username and password/token"
	case strings.Contains(errStr, "permission denied"):
		return "permission denied"
	default:
		return "an unexpected error occurred"
	}
}
