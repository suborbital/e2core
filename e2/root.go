package main

import (
	"github.com/spf13/cobra"

	"github.com/suborbital/e2core/e2/cli/command"
	"github.com/suborbital/e2core/e2/cli/features"
	"github.com/suborbital/e2core/e2/cli/release"
)

func rootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "e2",
		Short:   "Suborbital Extension Engine CLI",
		Version: release.Version(),
		Long:    `e2 is the full toolchain for using and managing the Suborbital Extension Engine (SE2).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Help()
			return nil
		},
	}

	create := &cobra.Command{
		Use:   "create",
		Short: "create a runnable, project, or handler",
		Long:  `create a new Atmo project, WebAssembly runnable or handler`,
	}

	create.AddCommand(command.CreateProjectCmd())
	create.AddCommand(command.CreateModuleCmd())
	// TODO: turn into create workflow command
	// Ref: https://github.com/suborbital/subo/issues/347
	// create.AddCommand(command.CreateHandlerCmd()).

	cmd.AddCommand(docsCommand())
	cmd.AddCommand(command.BuildCmd())
	cmd.AddCommand(command.CleanCmd())
	// TODO: Re-enable when dev is updated to work with e2core
	// cmd.AddCommand(command.DevCmd())

	// se2 related commands.
	create.AddCommand(command.SE2CreateTokenCommand())
	cmd.AddCommand(command.SE2DeployCommand())

	// experimental hidden commands
	if features.EnableReleaseCommands {
		create.AddCommand(command.CreateReleaseCmd())
	}

	if features.EnableRegistryCommands {
		cmd.AddCommand(command.PushCmd())

		// TODO: figure out how not to clash with the se2 deploy commsnd
		// cmd.AddCommand(command.DeployCmd())
	}

	cmd.AddCommand(create)
	cmd.SetVersionTemplate("e2 CLI v{{.Version}}\n")

	return cmd
}

func docsCommand() *cobra.Command {
	docs := &cobra.Command{
		Use:   "docs",
		Short: "documentation generation resources",
		Long:  "test and generate code embedded markdown documentation",
	}

	docs.AddCommand(command.DocsBuildCmd())
	docs.AddCommand(command.DocsTestCmd())

	return docs
}
