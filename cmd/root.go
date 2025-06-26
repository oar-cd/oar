// Package cmd implements the command line interface for Oar.
package cmd

import (
	"log"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/ch00k/oar/internal/app"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

var (
	database *gorm.DB
	dataDir  string
	verbose  bool
)

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
	rootCmd.PersistentFlags().StringVar(&dataDir, "data-dir", defaultDataDir, "Data directory for Oar")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")
}

func initLogging() {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})

	logger := slog.New(handler)
	slog.SetDefault(logger)
}

func GetDB() *gorm.DB {
	return database
}
