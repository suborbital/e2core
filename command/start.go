package command

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/suborbital/deltav/orchestrator"
	"github.com/suborbital/deltav/server"
	"github.com/suborbital/deltav/server/options"
	"github.com/suborbital/deltav/server/release"
	"github.com/suborbital/deltav/signaler"
	"github.com/suborbital/vektor/vlog"
)

type deltavInfo struct {
	DeltavVersion string `json:"deltav_version"`
}

func Start() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "start [bundle-path]",
		Short:   "start the deltav server",
		Long:    "starts the deltav server using the provided options",
		Version: release.DeltavServerDotVersion,
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "./runnables.wasm.zip"
			if len(args) > 0 {
				path = args[0]
			}

			logger := vlog.Default(
				vlog.AppMeta(deltavInfo{DeltavVersion: release.DeltavServerDotVersion}),
				vlog.EnvPrefix("DELTAV"),
			)

			opts, err := optionsFromFlags(cmd.Flags())
			if err != nil {
				return errors.Wrap(err, "failed to optionsFromFlags")
			}

			opts = append(
				opts,
				options.UseLogger(logger),
				options.UseBundlePath(path),
			)

			server, err := server.New(opts...)
			if err != nil {
				return errors.Wrap(err, "server.New")
			}

			orchestrator, err := orchestrator.New(path)
			if err != nil {
				return errors.Wrap(err, "failed to orchestrator.New")
			}

			if partnerCmd, _ := cmd.Flags().GetString(runPartnerFlag); partnerCmd != "" {
				if err := orchestrator.RunPartner(partnerCmd); err != nil {
					return errors.Wrap(err, "failed to RunPartner")
				}
			}

			signaler := signaler.Setup()

			signaler.Start(orchestrator.Start)

			signaler.Start(server.Start)

			return signaler.Wait(time.Second * 5)
		},
	}

	cmd.SetVersionTemplate("{{.Version}}\n")

	cmd.Flags().Bool(waitFlag, false, "if passed, DeltaV will wait until a bundle becomes available on disk, checking once per second")
	cmd.Flags().String(appNameFlag, "DeltaV", "if passed, it'll be used as DELTAV_APP_NAME, otherwise 'DeltaV' will be used")
	cmd.Flags().String(runPartnerFlag, "", "if passed, the provided command will be run as the partner application")
	cmd.Flags().String(domainFlag, "", "if passed, it'll be used as DELTAV_DOMAIN and HTTPS will be used, otherwise HTTP will be used")
	cmd.Flags().Int(httpPortFlag, 8080, "if passed, it'll be used as DELTAV_HTTP_PORT, otherwise '8080' will be used")
	cmd.Flags().Int(tlsPortFlag, 443, "if passed, it'll be used as DELTAV_TLS_PORT, otherwise '443' will be used")

	return cmd
}

func optionsFromFlags(flags *pflag.FlagSet) ([]options.Modifier, error) {
	appName, err := flags.GetString(appNameFlag)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("get string flag '%s' value", appNameFlag))
	}

	domain, err := flags.GetString(domainFlag)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("get string flag '%s' value", domainFlag))
	}

	httpPort, err := flags.GetInt(httpPortFlag)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("get int flag '%s' value", httpPortFlag))
	}

	tlsPort, err := flags.GetInt(tlsPortFlag)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("get int flag '%s' value", tlsPortFlag))
	}

	shouldWait := flags.Changed(waitFlag)
	shouldRunHeadless := flags.Changed(headlessFlag)

	opts := []options.Modifier{
		options.ShouldRunHeadless(shouldRunHeadless),
		options.ShouldWait(shouldWait),
		options.AppName(appName),
		options.Domain(domain),
		options.HTTPPort(httpPort),
		options.TLSPort(tlsPort),
	}

	return opts, nil
}
