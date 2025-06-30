// Package cmd implements the command line interface for Oar.
package cmd

import (
	"log"
	"os"
	"path/filepath"

	"github.com/ch00k/oar/internal/app"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

var (
	database      *gorm.DB
	dataDir       string
	colorDisabled bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "oar",
	Short: "GitOps for Docker Compose",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Set up colored output
		initColors()

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
	rootCmd.PersistentFlags().StringVarP(&dataDir, "data-dir", "d", defaultDataDir, "Data directory")
	rootCmd.PersistentFlags().VarP(logLevel, "log-level", "l", "Log level (debug, info, warning, error)")
	rootCmd.PersistentFlags().BoolVarP(&colorDisabled, "no-color", "c", false, "Disable colored output")
}

func GetDB() *gorm.DB {
	return database
}
