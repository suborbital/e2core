package command

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/suborbital/e2core/e2core"
	"github.com/suborbital/e2core/e2core/satbackend"
	"github.com/suborbital/e2core/options"
	"github.com/suborbital/e2core/server"
	"github.com/suborbital/e2core/server/release"
	"github.com/suborbital/vektor/vlog"
)

type e2coreInfo struct {
	E2CoreVersion string `json:"e2core_version"`
	ModuleName    string `json:"module_name,omitempty"`
}

func Start() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "start [bundle-path]",
		Short:   "start the e2core server",
		Long:    "starts the e2core server using the provided options",
		Version: release.E2CoreServerDotVersion,
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "./modules.wasm.zip"
			if len(args) > 0 {
				path = args[0]
			}

			logger := vlog.Default(
				vlog.AppMeta(e2coreInfo{E2CoreVersion: release.E2CoreServerDotVersion}),
				vlog.EnvPrefix("E2CORE"),
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

			backend, err := satbackend.New(server.Options(), server.Syncer())
			if err != nil {
				return errors.Wrap(err, "failed to satbackend.New")
			}

			system := e2core.NewSystem(server, backend)

			system.StartAll()

			return system.ShutdownWait()
		},
	}

	cmd.SetVersionTemplate("{{.Version}}\n")

	cmd.Flags().String(domainFlag, "", "if passed, it'll be used as E2CORE_DOMAIN and HTTPS will be used, otherwise HTTP will be used")
	cmd.Flags().Int(httpPortFlag, 8080, "if passed, it'll be used as E2CORE_HTTP_PORT, otherwise '8080' will be used")
	cmd.Flags().Int(tlsPortFlag, 443, "if passed, it'll be used as E2CORE_TLS_PORT, otherwise '443' will be used")

	return cmd
}

func optionsFromFlags(flags *pflag.FlagSet) ([]options.Modifier, error) {
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

	opts := []options.Modifier{
		options.Domain(domain),
		options.HTTPPort(httpPort),
		options.TLSPort(tlsPort),
	}

	return opts, nil
}
