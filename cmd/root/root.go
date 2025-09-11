// Package root implements the command line interface for Oar.
package root

import (
	"log"
	"os"

	"github.com/oar-cd/oar/cmd/logs"
	"github.com/oar-cd/oar/cmd/output"
	"github.com/oar-cd/oar/cmd/project"
	"github.com/oar-cd/oar/cmd/start"
	"github.com/oar-cd/oar/cmd/status"
	"github.com/oar-cd/oar/cmd/stop"
	"github.com/oar-cd/oar/cmd/update"
	"github.com/oar-cd/oar/cmd/version"
	"github.com/oar-cd/oar/internal/app"
	"github.com/oar-cd/oar/logging"
	"github.com/oar-cd/oar/services"
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
			skipInitCommands := []string{"version", "update", "logs", "start", "stop", "status"}

			// For root-level commands (parent is "oar"), skip initialization
			if cmd.Parent() != nil && cmd.Parent().Name() == "oar" {
				for _, skipCmd := range skipInitCommands {
					if cmd.Name() == skipCmd {
						return
					}
				}
			}

			// Initialize configuration for CLI with data directory override
			// Only override if the flag was explicitly set by the user (differs from default)
			var cliDataDirOverride string
			if dataDir != defaultDataDir {
				// CLI flag was provided and differs from default, use it to override
				cliDataDirOverride = dataDir
			}
			// Otherwise pass empty string to let config system use environment variables/defaults

			var err error
			config, err = services.NewConfigForCLI(cliDataDirOverride)
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

	cmd.AddCommand(logs.NewCmdLogs())
	cmd.AddCommand(project.NewCmdProject())
	cmd.AddCommand(start.NewCmdStart())
	cmd.AddCommand(status.NewCmdStatus())
	cmd.AddCommand(stop.NewCmdStop())
	cmd.AddCommand(update.NewCmdUpdate())
	cmd.AddCommand(version.NewCmdVersion())
	return cmd
}
