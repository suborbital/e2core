package main

import (
	"github.com/spf13/cobra"

	"github.com/suborbital/deltav/command"
	"github.com/suborbital/deltav/server/release"
)

func main() {
	root := rootCommand()
	root.AddCommand(command.Start())

	root.Execute()
}

func rootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "deltav",
		Version: release.DeltavServerDotVersion,
		Long: `
	Deltav is a secure development kit and server for writing and running untrusted third-party plugins.
	
	The DeltaV server is responsible for managing and running plugins using simple HTTP, RPC, or streaming interfaces.`,
	}

	return cmd
}
