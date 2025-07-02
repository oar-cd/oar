// Package logging provides logging utilities for Oar, allowing configuration of log levels and output formats.
package logging

import (
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"
)

var LogLevel = newLogLevelValue("info", []string{"debug", "info", "warning", "error"})

type logLevelValue struct {
	value   string
	allowed []string
}

func newLogLevelValue(defaultValue string, allowed []string) *logLevelValue {
	return &logLevelValue{
		value:   defaultValue,
		allowed: allowed,
	}
}

func (l *logLevelValue) Set(value string) error {
	if slices.Contains(l.allowed, value) {
		l.value = value
		return nil
	}

	return fmt.Errorf("invalid value '%s'. Allowed values: %s",
		value, strings.Join(l.allowed, ", "))
}

func (l *logLevelValue) String() string {
	return l.value
}

func (l *logLevelValue) Type() string {
	return fmt.Sprintf("one of [%s]", strings.Join(l.allowed, "|"))
}

func (l *logLevelValue) slogValue() slog.Level {
	switch l.value {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo // Default to info if something goes wrong
	}
}

func InitLogging() {
	level := LogLevel.slogValue()

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})

	logger := slog.New(handler)
	slog.SetDefault(logger)
}
