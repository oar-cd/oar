// Package version provides the version command for Oar.
package version

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Build-time variables (set via -ldflags)
var (
	Version = "dev" // Version of the Oar binary
)

// NewCmdVersion creates the version command
func NewCmdVersion() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Long:  `Display version information for Oar.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVersion()
		},
	}

	return cmd
}

func runVersion() error {
	fmt.Println(Version)
	return nil
}
