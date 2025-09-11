// Package version provides the version command for Oar.
package version

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/oar-cd/oar/cmd/output"
	"github.com/oar-cd/oar/services"
	"github.com/spf13/cobra"
)

// Build-time variables (set via -ldflags)
var (
	CLIVersion = "dev" // Version of the CLI binary
)

// NewCmdVersion creates the version command
func NewCmdVersion() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Long:  `Display version information for both the CLI binary and installation.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVersion()
		},
	}

	return cmd
}

func runVersion() error {
	// CLI Binary version info
	fmt.Print(output.PrintMessage(output.Plain, "CLI version: %s", CLIVersion))

	// Installation version (if available)
	serverVersion := getServerVersion()
	if serverVersion != "" {
		fmt.Print(output.PrintMessage(output.Plain, "Server version: %s", serverVersion))
	}

	return nil
}

// getServerVersion reads the VERSION file from the installation directory
func getServerVersion() string {
	oarDir := services.GetDefaultDataDir()
	versionFile := filepath.Join(oarDir, "VERSION")

	data, err := os.ReadFile(versionFile)
	if err != nil {
		return "unknown" // Installation version not available
	}

	return strings.TrimSpace(string(data))
}
