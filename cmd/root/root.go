// Package root implements the command line interface for Oar.
package root

import (
	"log"
	"os"

	"github.com/ch00k/oar/cmd/output"
	"github.com/ch00k/oar/cmd/project"
	"github.com/ch00k/oar/cmd/update"
	"github.com/ch00k/oar/cmd/version"
	"github.com/ch00k/oar/internal/app"
	"github.com/ch00k/oar/logging"
	"github.com/ch00k/oar/services"
	"github.com/spf13/cobra"
)

var config *services.Config

func Execute() {
	defaultDataDir := services.GetDefaultDataDir()

	if err := NewCmdRoot(defaultDataDir).Execute(); err != nil {
		os.Exit(1)
	}
}

func NewCmdRoot(defaultDataDir string) *cobra.Command {
	var dataDir string

	cmd := &cobra.Command{
		Use:   "oar",
		Short: "GitOps deployment tool for Docker Compose projects",
		Long: `Oar manages Docker Compose applications deployed from Git repositories.
	It handles cloning, updates, and deployments with full state tracking.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Skip initialization for commands that don't need full app context
			if cmd.Name() == "version" || cmd.Name() == "update" {
				return
			}

			// Initialize configuration for CLI with data directory override
			var err error
			config, err = services.NewConfigForCLI(dataDir)
			if err != nil {
				log.Fatalf("Failed to initialize configuration: %s", err)
				os.Exit(1)
			}

			// Initialize colors (CLI flag overrides config)
			colorDisabled := !config.ColorEnabled
			if output.NoColor.IsSet() {
				colorDisabled = true // --no-color flag overrides config
			}
			output.InitColors(colorDisabled)

			// Initialize logging (CLI flag overrides config)
			logLevel := config.LogLevel
			if logging.LogLevel.IsSet() {
				logLevel = logging.LogLevel.String()
			}
			logging.InitLogging(logLevel)

			// Initialize application with config
			if err := app.InitializeWithConfig(config); err != nil {
				log.Fatalf("Failed to initialize application: %s", err)
				os.Exit(1)
			}
		},
	}

	cmd.PersistentFlags().
		StringVarP(&dataDir, "data-dir", "d", defaultDataDir, "Data directory for Oar configuration and projects")
	cmd.PersistentFlags().VarP(logging.LogLevel, "log-level", "l", "Set log verbosity level")
	cmd.PersistentFlags().VarP(output.NoColor, "no-color", "c", "Disable colored terminal output")

	cmd.AddCommand(project.NewCmdProject())
	cmd.AddCommand(update.NewCmdUpdate())
	cmd.AddCommand(version.NewCmdVersion())
	return cmd
}
