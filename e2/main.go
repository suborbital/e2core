package main

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/suborbital/e2core/e2/command"
	"github.com/suborbital/e2core/e2/util"
	"github.com/suborbital/e2core/e2core/release"
)

func main() {
	cmd := &cobra.Command{
		Use:     "e2",
		Short:   "E2 Core CLI",
		Version: release.E2CoreServerDotVersion,
		Long:    `e2 is a simple deployment tool to manage your E2 Core Kubernetes deployments.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Help()
			return nil
		},
	}

	cmd.SetVersionTemplate("E2 Core deployment CLI v{{.Version}}\n")

	cmd.AddCommand(command.DeployCommand())
	cmd.AddCommand(command.StatusCommand())

	if err := cmd.Execute(); err != nil {
		util.LogFail(err.Error())
		os.Exit(1)
	}
}
