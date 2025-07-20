package services

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
	case strings.Contains(errStr, "permission denied"):
		return "permission denied"
	case strings.Contains(errStr, "connection"):
		return "database connection failed"
	case strings.Contains(errStr, "timeout"):
		return "operation timed out"
	default:
		return "an unexpected error occurred"
	}
}
