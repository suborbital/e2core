package main

import (
	"os"
)

func main() {
	rootCmd := rootCommand()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}

	checkForUpdates()
}
