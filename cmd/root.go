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
	Short: "GitOps deployment tool for Docker Compose projects",
	Long: `Oar manages Docker Compose applications deployed from Git repositories.
It handles cloning, updates, and deployments with full state tracking.`,
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
	rootCmd.PersistentFlags().StringVarP(&dataDir, "data-dir", "d", defaultDataDir, "Data directory for Oar configuration and projects")
	rootCmd.PersistentFlags().VarP(logLevel, "log-level", "l", "Set log verbosity level")
	rootCmd.PersistentFlags().BoolVarP(&colorDisabled, "no-color", "c", false, "Disable colored terminal output")
}

func GetDB() *gorm.DB {
	return database
}
