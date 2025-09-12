// Package project provides commands for managing Docker Compose projects in Oar.
package project

import "github.com/spf13/cobra"

func NewCmdProject() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Manage Docker Compose projects",
	}

	cmd.AddCommand(NewCmdProjectList())
	cmd.AddCommand(NewCmdProjectAdd())
	cmd.AddCommand(NewCmdProjectRemove())
	cmd.AddCommand(NewCmdProjectShow())
	cmd.AddCommand(NewCmdProjectDeploy())
	cmd.AddCommand(NewCmdProjectStop())
	cmd.AddCommand(NewCmdProjectStatus())
	cmd.AddCommand(NewCmdProjectConfig())
	cmd.AddCommand(NewCmdProjectLogs())
	cmd.AddCommand(NewCmdProjectDeployments())
	return cmd
}
