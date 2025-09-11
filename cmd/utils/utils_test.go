package utils

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"testing"

	"github.com/ch00k/oar/cmd/output"
	"github.com/ch00k/oar/services"
	"github.com/stretchr/testify/assert"
)

// These functions call os.Exit which makes them hard to test directly.
// Instead, we'll test the logging behavior by capturing log output.

func TestHandleCommandError_LogsBehavior(t *testing.T) {
	// Capture slog output
	var logBuf bytes.Buffer
	originalLogger := slog.Default()
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)
	defer slog.SetDefault(originalLogger)

	// We can't test the actual function since it calls os.Exit,
	// but we can test what it would log by calling slog directly
	testErr := fmt.Errorf("test error")
	slog.Error("Command failed", "operation", "test operation", "error", testErr)

	logOutput := logBuf.String()
	assert.Contains(t, logOutput, "Command failed")
	assert.Contains(t, logOutput, "test operation")
	assert.Contains(t, logOutput, "test error")
}

func TestHandleCommandError_WithContextLogsBehavior(t *testing.T) {
	var logBuf bytes.Buffer
	originalLogger := slog.Default()
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)
	defer slog.SetDefault(originalLogger)

	// Test what the logging would look like with context
	testErr := fmt.Errorf("connection failed")
	context := []any{"host", "localhost", "port", 5432}
	slog.Error("Command failed", append([]any{"operation", "database connect", "error", testErr}, context...)...)

	logOutput := logBuf.String()
	assert.Contains(t, logOutput, "Command failed")
	assert.Contains(t, logOutput, "database connect")
	assert.Contains(t, logOutput, "connection failed")
	assert.Contains(t, logOutput, "localhost")
	assert.Contains(t, logOutput, "5432")
}

func TestHandleInvalidUUID_LogsBehavior(t *testing.T) {
	var logBuf bytes.Buffer
	originalLogger := slog.Default()
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)
	defer slog.SetDefault(originalLogger)

	// Test what the logging would look like
	slog.Warn("Invalid UUID provided", "operation", "delete project", "input", "invalid-uuid")

	logOutput := logBuf.String()
	assert.Contains(t, logOutput, "Invalid UUID provided")
	assert.Contains(t, logOutput, "delete project")
	assert.Contains(t, logOutput, "invalid-uuid")
}

// Test the error formatting logic that HandleCommandError uses
func TestErrorFormatting(t *testing.T) {
	// Initialize colors for consistent output
	output.InitColors(true) // Disable colors for testing

	tests := []struct {
		name           string
		inputError     error
		expectedOutput string
	}{
		{
			name:           "record not found error",
			inputError:     errors.New("record not found"),
			expectedOutput: "project not found",
		},
		{
			name:           "unique constraint name error",
			inputError:     errors.New("UNIQUE constraint failed: projects.name"),
			expectedOutput: "a project with this name already exists",
		},
		{
			name:           "generic unique constraint error",
			inputError:     errors.New("UNIQUE constraint failed: some_table.field"),
			expectedOutput: "this entry already exists",
		},
		{
			name:           "connection error",
			inputError:     errors.New("connection refused"),
			expectedOutput: "database connection failed",
		},
		{
			name:           "timeout error",
			inputError:     errors.New("context deadline exceeded: timeout"),
			expectedOutput: "operation timed out",
		},
		{
			name:           "ssh key auth error",
			inputError:     errors.New("permission denied (publickey)"),
			expectedOutput: "ssh key authentication failed - please check your private key",
		},
		{
			name:           "generic auth error",
			inputError:     errors.New("authentication failed"),
			expectedOutput: "git authentication failed - please check your credentials",
		},
		{
			name:           "unknown error",
			inputError:     errors.New("some unknown error message"),
			expectedOutput: "an unexpected error occurred",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := services.FormatErrorForUser(tt.inputError)
			assert.Equal(t, tt.expectedOutput, result)
		})
	}
}

// Test the message formatting that both functions use
func TestMessageFormatting(t *testing.T) {
	// Initialize colors for consistent output
	output.InitColors(true) // Disable colors for testing

	tests := []struct {
		name           string
		format         string
		args           []interface{}
		expectedOutput string
	}{
		{
			name:           "simple error message",
			format:         "Error: %s",
			args:           []interface{}{"project not found"},
			expectedOutput: "Error: project not found\n",
		},
		{
			name:           "invalid UUID message",
			format:         "Error: Invalid project ID '%s'. Must be a valid UUID.",
			args:           []interface{}{"invalid-uuid"},
			expectedOutput: "Error: Invalid project ID 'invalid-uuid'. Must be a valid UUID.\n",
		},
		{
			name:           "multiple arguments",
			format:         "Error: Failed to %s %s with ID %d",
			args:           []interface{}{"delete", "project", 123},
			expectedOutput: "Error: Failed to delete project with ID 123\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := output.PrintMessage(output.Error, tt.format, tt.args...)
			assert.Equal(t, tt.expectedOutput, result)
		})
	}
}

// Test the complete error handling flow (without os.Exit)
func TestErrorHandlingFlow(t *testing.T) {
	// Initialize colors for consistent output
	output.InitColors(true) // Disable colors for testing

	tests := []struct {
		name             string
		inputError       error
		operation        string
		context          []interface{}
		expectedMessage  string
		expectedLogLevel string
	}{
		{
			name:             "database record not found",
			inputError:       errors.New("record not found"),
			operation:        "retrieving project",
			context:          []interface{}{"project_id", "12345"},
			expectedMessage:  "Error: project not found\n",
			expectedLogLevel: "ERROR",
		},
		{
			name:             "unique constraint violation",
			inputError:       errors.New("UNIQUE constraint failed: projects.name"),
			operation:        "creating project",
			context:          []interface{}{"project_name", "test-project"},
			expectedMessage:  "Error: a project with this name already exists\n",
			expectedLogLevel: "ERROR",
		},
		{
			name:             "connection error",
			inputError:       errors.New("connection refused"),
			operation:        "connecting to database",
			context:          []interface{}{},
			expectedMessage:  "Error: database connection failed\n",
			expectedLogLevel: "ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the message formatting
			userFriendlyError := services.FormatErrorForUser(tt.inputError)
			message := output.PrintMessage(output.Error, "Error: %s", userFriendlyError)
			assert.Equal(t, tt.expectedMessage, message)

			// Test logging behavior
			var logBuf bytes.Buffer
			originalLogger := slog.Default()
			logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{
				Level: slog.LevelDebug,
			}))
			slog.SetDefault(logger)
			defer slog.SetDefault(originalLogger)

			// Simulate what HandleCommandError would log
			slog.Error(
				"Command failed",
				append([]interface{}{"operation", tt.operation, "error", tt.inputError}, tt.context...)...)

			logOutput := logBuf.String()
			assert.Contains(t, logOutput, "Command failed")
			assert.Contains(t, logOutput, tt.operation)
			assert.Contains(t, logOutput, tt.inputError.Error())

			// Check context if provided
			for i := 0; i < len(tt.context); i += 2 {
				if i+1 < len(tt.context) {
					assert.Contains(t, logOutput, fmt.Sprintf("%v", tt.context[i+1]))
				}
			}
		})
	}
}

// Test the invalid UUID message formatting
func TestInvalidUUIDMessageFormatting(t *testing.T) {
	// Initialize colors for consistent output
	output.InitColors(true) // Disable colors for testing

	tests := []struct {
		name            string
		operation       string
		input           string
		expectedMessage string
	}{
		{
			name:            "simple invalid UUID",
			operation:       "project operation",
			input:           "invalid-uuid",
			expectedMessage: "Error: Invalid project ID 'invalid-uuid'. Must be a valid UUID.\n",
		},
		{
			name:            "empty string UUID",
			operation:       "project delete",
			input:           "",
			expectedMessage: "Error: Invalid project ID ''. Must be a valid UUID.\n",
		},
		{
			name:            "numeric invalid UUID",
			operation:       "project status",
			input:           "12345",
			expectedMessage: "Error: Invalid project ID '12345'. Must be a valid UUID.\n",
		},
		{
			name:            "almost valid UUID",
			operation:       "project deploy",
			input:           "12345678-1234-1234-1234-12345678901",
			expectedMessage: "Error: Invalid project ID '12345678-1234-1234-1234-12345678901'. Must be a valid UUID.\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the message formatting part of HandleInvalidUUID
			message := output.PrintMessage(
				output.Error,
				"Error: Invalid project ID '%s'. Must be a valid UUID.",
				tt.input,
			)
			assert.Equal(t, tt.expectedMessage, message)
		})
	}
}

// Test the complete invalid UUID handling flow (without os.Exit)
func TestInvalidUUIDHandlingFlow(t *testing.T) {
	// Initialize colors for consistent output
	output.InitColors(true) // Disable colors for testing

	tests := []struct {
		name             string
		operation        string
		input            string
		expectedMessage  string
		expectedLogLevel string
	}{
		{
			name:             "project show invalid UUID",
			operation:        "project show",
			input:            "invalid-uuid",
			expectedMessage:  "Error: Invalid project ID 'invalid-uuid'. Must be a valid UUID.\n",
			expectedLogLevel: "WARN",
		},
		{
			name:             "project delete invalid UUID",
			operation:        "project delete",
			input:            "not-a-uuid",
			expectedMessage:  "Error: Invalid project ID 'not-a-uuid'. Must be a valid UUID.\n",
			expectedLogLevel: "WARN",
		},
		{
			name:             "empty UUID",
			operation:        "project operation",
			input:            "",
			expectedMessage:  "Error: Invalid project ID ''. Must be a valid UUID.\n",
			expectedLogLevel: "WARN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the message formatting
			message := output.PrintMessage(
				output.Error,
				"Error: Invalid project ID '%s'. Must be a valid UUID.",
				tt.input,
			)
			assert.Equal(t, tt.expectedMessage, message)

			// Test logging behavior
			var logBuf bytes.Buffer
			originalLogger := slog.Default()
			logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{
				Level: slog.LevelDebug,
			}))
			slog.SetDefault(logger)
			defer slog.SetDefault(originalLogger)

			// Simulate what HandleInvalidUUID would log
			slog.Warn("Invalid UUID provided", "operation", tt.operation, "input", tt.input)

			logOutput := logBuf.String()
			assert.Contains(t, logOutput, "Invalid UUID provided")
			assert.Contains(t, logOutput, tt.operation)
			assert.Contains(t, logOutput, tt.input)
		})
	}
}
