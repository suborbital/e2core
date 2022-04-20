package main

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/suborbital/atmo/atmo"
	"github.com/suborbital/atmo/atmo/options"
	"github.com/suborbital/atmo/atmo/release"
	"github.com/suborbital/vektor/vlog"
)

const (
	headlessFlag = "headless"
	waitFlag     = "wait"
	appNameFlag  = "appName"
	domainFlag   = "domain"
	httpPortFlag = "httpPort"
	tlsPortFlag  = "tlsPort"
)

type atmoInfo struct {
	AtmoVersion string `json:"atmo_version"`
}

func main() {
	cmd := rootCommand()
	cmd.Execute()
}

func rootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "atmo [bundle-path]",
		Short: "Atmo function-based web service runner",
		Long: `
Atmo is an all-in-one function-based web service platform that enables 
building backend systems using composable WebAssembly modules in a declarative manner.

Atmo automatically scales using a meshed message bus, job scheduler, and 
flexible API gateway to handle any workload. 

Handling API and event-based traffic is made simple using the declarative 
Directive format and the powerful Runnable API using a variety of languages.`,
		Version: release.AtmoDotVersion,
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "./runnables.wasm.zip"
			if len(args) > 0 {
				path = args[0]
			}

			logger := vlog.Default(
				vlog.AppMeta(atmoInfo{AtmoVersion: release.AtmoDotVersion}),
				vlog.Level(vlog.LogLevelInfo),
				vlog.EnvPrefix("ATMO"),
			)

			appName, err := cmd.Flags().GetString(appNameFlag)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("get string flag '%s' value", appNameFlag))
			}

			domain, err := cmd.Flags().GetString(domainFlag)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("get string flag '%s' value", domainFlag))
			}

			httpPort, err := cmd.Flags().GetInt(httpPortFlag)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("get int flag '%s' value", httpPortFlag))
			}

			tlsPort, err := cmd.Flags().GetInt(tlsPortFlag)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("get int flag '%s' value", tlsPortFlag))
			}

			shouldWait := cmd.Flags().Changed(waitFlag)
			shouldRunHeadless := cmd.Flags().Changed(headlessFlag)

			atmoService, err := atmo.New(
				options.UseLogger(logger),
				options.UseBundlePath(path),
				options.ShouldRunHeadless(shouldRunHeadless),
				options.ShouldWait(shouldWait),
				options.AppName(appName),
				options.Domain(domain),
				options.HTTPPort(httpPort),
				options.TLSPort(tlsPort),
			)
			if err != nil {
				return errors.Wrap(err, "atmo.New")
			}

			return atmoService.Start()
		},
	}

	cmd.SetVersionTemplate("{{.Version}}\n")

	cmd.Flags().Bool(waitFlag, false, "if passed, Atmo will wait until a bundle becomes available on disk, checking once per second")
	cmd.Flags().String(appNameFlag, "Atmo", "if passed, it'll be used as ATMO_APP_NAME, otherwise 'Atmo' will be used")
	cmd.Flags().String(domainFlag, "", "if passed, it'll be used as ATMO_DOMAIN and HTTPS will be used, otherwise HTTP will be used")
	cmd.Flags().Int(httpPortFlag, 8080, "if passed, it'll be used as ATMO_HTTP_PORT, otherwise '8080' will be used")
	cmd.Flags().Int(tlsPortFlag, 443, "if passed, it'll be used as ATMO_TLS_PORT, otherwise '443' will be used")

	return cmd
}
