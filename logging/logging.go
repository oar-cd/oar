// Package logging provides logging utilities for Oar.
package logging

import (
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"
)

// ParseLogLevel converts a string log level to slog.Level
func ParseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	case "silent", "none":
		// Return a very high level to effectively disable all logging
		return slog.Level(1000)
	default:
		return slog.LevelInfo
	}
}

// ValidLogLevels returns the list of valid log levels
func ValidLogLevels() []string {
	return []string{"debug", "info", "warning", "error", "silent"}
}

// InitLogging initializes logging with the specified log level
func InitLogging(logLevel string) {
	level := ParseLogLevel(logLevel)

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})

	logger := slog.New(handler)
	slog.SetDefault(logger)
}

// CLI flag for setting the log level

// LogLevel is a flag for setting the log level
var LogLevel = &logLevelFlag{value: "silent", set: false}

type logLevelFlag struct {
	value string
	set   bool
}

func (l *logLevelFlag) Set(value string) error {
	if !slices.Contains(ValidLogLevels(), value) {
		return fmt.Errorf("invalid value '%s'. Allowed values: %s",
			value, strings.Join(ValidLogLevels(), ", "))
	}
	l.value = value
	l.set = true
	return nil
}

func (l *logLevelFlag) String() string {
	return l.value
}

func (l *logLevelFlag) Type() string {
	return fmt.Sprintf("one of [%s]", strings.Join(ValidLogLevels(), "|"))
}

// IsSet returns true if the flag was explicitly set via command line
func (l *logLevelFlag) IsSet() bool {
	return l.set
}
