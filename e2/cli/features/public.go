//go:build !development
// +build !development

package features

// EnableReleaseCommands and others are feature flags.
const (
	EnableReleaseCommands  = false
	EnableRegistryCommands = false
)
