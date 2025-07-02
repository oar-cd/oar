// Package utils provides utility functions for CLI commands in Oar.
package utils

import (
	"log/slog"
	"os"

	"github.com/ch00k/oar/cmd/output"
)

// HandleCommandError provides consistent error handling for CLI commands
func HandleCommandError(operation string, err error, context ...any) {
	slog.Error("Command failed", append([]any{"operation", operation, "error", err}, context...)...)
	output.PrintMessage(output.Error, "Error: %s failed: %v", operation, err)
	os.Exit(1)
}

// HandleInvalidUUID provides consistent handling for invalid UUID errors
func HandleInvalidUUID(operation, input string) {
	slog.Warn("Invalid UUID provided", "operation", operation, "input", input)
	output.PrintMessage(output.Error, "Error: Invalid project ID '%s'. Must be a valid UUID.", input)
	os.Exit(1)
}
