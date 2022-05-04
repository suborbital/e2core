package main

import (
	"github.com/spf13/cobra"

	"github.com/suborbital/velocity/command"
	"github.com/suborbital/velocity/server/release"
)

func main() {
	root := rootCommand()
	root.AddCommand(command.Start())

	root.Execute()
}

func rootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "velocity [bundle-path]",
		Version: release.VelocityServerDotVersion,
		Long: `
	Velocity is an all-in-one cloud native functions framework that enables 
	building backend systems using composable WebAssembly modules in a declarative manner.
	
	Velocity automatically extends any application with stateless, ephemeral functions that
	execute within a secure sandbox, written in any language. 
	
	Handling API and event-based traffic is made simple using the declarative 
	Directive format and the powerful API available for many languages.`,
	}

	return cmd
}
