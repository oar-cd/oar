// Package root implements the command line interface for Oar.
package root

import (
	"log"
	"os"
	"path/filepath"

	"github.com/ch00k/oar/cmd/output"
	"github.com/ch00k/oar/cmd/project"
	"github.com/ch00k/oar/internal/app"
	"github.com/ch00k/oar/logging"
	"github.com/spf13/cobra"
)

var isColorDisabled bool

func Execute() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Failed to get home directory: %s", err)
		os.Exit(1)
	}

	defaultDataDir := filepath.Join(homeDir, ".oar")

	err = NewCmdRoot(defaultDataDir).Execute()
	if err != nil {
		os.Exit(1)
	}
}

func NewCmdRoot(dataDir string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "oar",
		Short: "GitOps deployment tool for Docker Compose projects",
		Long: `Oar manages Docker Compose applications deployed from Git repositories.
	It handles cloning, updates, and deployments with full state tracking.`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			output.InitColors(isColorDisabled)

			if err := app.Initialize(dataDir); err != nil {
				log.Fatalf("Failed to initialize application: %s", err)
				os.Exit(1)
			}
		},
	}

	cmd.PersistentFlags().
		StringVarP(&dataDir, "data-dir", "d", dataDir, "Data directory for Oar configuration and projects")
	cmd.PersistentFlags().VarP(logging.LogLevel, "log-level", "l", "Set log verbosity level")
	cmd.PersistentFlags().BoolVarP(&isColorDisabled, "no-color", "c", false, "Disable colored terminal output")

	cmd.AddCommand(project.NewCmdProject())
	return cmd
}
