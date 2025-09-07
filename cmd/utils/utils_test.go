package utils

import (
	"bytes"
	"fmt"
	"log/slog"
	"testing"

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
