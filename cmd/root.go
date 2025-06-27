// Package cmd implements the command line interface for Oar.
package cmd

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/ch00k/oar/internal/app"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

var (
	database *gorm.DB
	dataDir  string
)

var logLevel = newLogLevelValue("info", []string{"debug", "info", "warning", "error"})

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
	for _, allowed := range l.allowed {
		if value == allowed {
			l.value = value
			return nil
		}
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

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "oar",
	Short: "GitOps for Docker Compose",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Initialize structured logging
		initLogging()

		// Initialize app context
		if err := app.Initialize(dataDir); err != nil {
			log.Fatalf("Failed to initialize: %s", err)
		}
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	homeDir, _ := os.UserHomeDir()
	defaultDataDir := filepath.Join(homeDir, ".oar")
	rootCmd.PersistentFlags().StringVarP(&dataDir, "data-dir", "d", defaultDataDir, "Data directory for Oar")
	rootCmd.PersistentFlags().VarP(logLevel, "log-level", "l", "Log level (debug, info, warning, error)")
}

func initLogging() {
	level := logLevel.slogValue()

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})

	logger := slog.New(handler)
	slog.SetDefault(logger)
}

func GetDB() *gorm.DB {
	return database
}
