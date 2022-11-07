package sat

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/sethvargo/go-envconfig"
	"gopkg.in/yaml.v2"

	"github.com/suborbital/appspec/capabilities"
	"github.com/suborbital/appspec/fqmn"
	"github.com/suborbital/appspec/system"
	"github.com/suborbital/appspec/system/client"
	"github.com/suborbital/appspec/tenant"
	"github.com/suborbital/e2core/fqfn"
	"github.com/suborbital/e2core/options"
	"github.com/suborbital/vektor/vlog"

	satOptions "github.com/suborbital/e2core/sat/sat/options"
)

var useStdin bool

func init() {
	flag.BoolVar(&useStdin, "stdin", false, "read stdin as input, return output to stdout and then terminate")
}

type Config struct {
	RunnableArg     string
	JobType         string
	PrettyName      string
	Module          *tenant.Module
	Identifier      string
	CapConfig       capabilities.CapabilityConfig
	Port            int
	UseStdin        bool
	ControlPlaneUrl string
	EnvToken        string
	Logger          *vlog.Logger
	ProcUUID        string
	TracerConfig    satOptions.TracerConfig
	MetricsConfig   satOptions.MetricsConfig
}

type satInfo struct {
	SatVersion string `json:"sat_version"`
}

type app struct {
	Name string `json:"name"`
}

func ConfigFromArgs() (*Config, error) {
	flag.Parse()
	args := flag.Args()

	if len(args) < 1 {
		return nil, errors.New("missing argument: module (path, URL or FQMN)")
	}

	runnableArg := args[0]

	return ConfigFromRunnableArg(runnableArg)
}

func ConfigFromRunnableArg(runnableArg string) (*Config, error) {
	logger := vlog.Default(
		vlog.EnvPrefix("SAT"),
		vlog.AppMeta(satInfo{SatVersion: SatDotVersion}),
	)

	var module *tenant.Module

	opts, err := satOptions.Resolve(envconfig.OsLookuper())
	if err != nil {
		return nil, errors.Wrap(err, "configFromRunnableArg options.Resolve")
	}

	// first, determine if we need to connect to a control plane
	controlPlane := ""
	useControlPlane := false
	if opts.ControlPlane != nil {
		controlPlane = opts.ControlPlane.Address
		useControlPlane = true
	}

	appClient := client.NewHTTPSource(controlPlane, NewAuthToken(opts.EnvToken))
	caps := capabilities.DefaultConfigWithLogger(logger)

	if useControlPlane {
		opts := options.NewWithModifiers(options.UseLogger(logger))

		if err = appClient.Start(opts); err != nil {
			return nil, errors.Wrap(err, "failed to systemSource.Start")
		}
	}

	// next, handle the module arg being a URL, an FQMN, or a path on disk
	if isURL(runnableArg) {
		logger.Debug("fetching module from URL")
		tmpFile, err := downloadFromURL(runnableArg)
		if err != nil {
			return nil, errors.Wrap(err, "failed to downloadFromURL")
		}

		runnableArg = tmpFile
	} else if FQMN, err := fqmn.Parse(runnableArg); err == nil {
		if useControlPlane {
			logger.Debug("fetching module from control plane")

			cpModule, err := appClient.GetModule(runnableArg)
			if err != nil {
				return nil, errors.Wrap(err, "failed to FindRunnable")
			}

			module = cpModule

			// TODO: find an appropriate value for the version parameter
			rendered, err := system.ResolveCapabilitiesFromSource(appClient, FQMN.Tenant, FQMN.Namespace, logger)
			if err != nil {
				return nil, errors.Wrap(err, "failed to capabilities.Render")
			}

			caps = *rendered
		}
	} else {
		diskRunnable, err := findModuleDotYaml(runnableArg)
		if err != nil {
			return nil, errors.Wrap(err, "failed to findRunnable")
		}

		if diskRunnable != nil {
			if opts.Ident != nil && opts.Version != nil {
				FQMN, err := fqmn.FromParts(opts.Ident.Data, module.Namespace, module.Name, opts.Version.Data)
				if err != nil {
					return nil, errors.Wrap(err, "failed to fqmn.FromParts")
				}

				module.FQMN = FQMN
			}
		}

		module = diskRunnable
	}

	// set some defaults in the case we're not running in an application
	portInt, _ := strconv.Atoi(string(opts.Port))
	jobType := strings.TrimSuffix(filepath.Base(runnableArg), ".wasm")
	FQMN := fqfn.Parse(jobType)
	prettyName := jobType

	// modify configuration if we ARE running as part of an application
	if module != nil && module.FQMN != "" {
		jobType = module.FQMN
		FQMN = fqfn.Parse(module.FQMN)

		prettyName = fmt.Sprintf("%s-%s", jobType, opts.ProcUUID[:6])

		// replace the logger with something more detailed
		logger = vlog.Default(
			vlog.EnvPrefix("SAT"),
			vlog.AppMeta(app{prettyName}),
		)

		logger.Debug("configuring", jobType)
		logger.Debug("joining app", FQMN.Identifier)
	} else {
		logger.Debug("configuring", jobType)
	}

	// finally, put it all together
	c := &Config{
		RunnableArg:     runnableArg,
		JobType:         jobType,
		PrettyName:      prettyName,
		Module:          module,
		Identifier:      FQMN.Identifier,
		CapConfig:       caps,
		Port:            portInt,
		UseStdin:        useStdin,
		ControlPlaneUrl: controlPlane,
		Logger:          logger,
		TracerConfig:    opts.TracerConfig,
		MetricsConfig:   opts.MetricsConfig,
		ProcUUID:        string(opts.ProcUUID),
	}

	return c, nil
}

func findModuleDotYaml(runnableArg string) (*tenant.Module, error) {
	filename := filepath.Base(runnableArg)
	moduleFilepath := strings.Replace(runnableArg, filename, ".module.yml", -1)

	if _, err := os.Stat(moduleFilepath); err != nil {
		// .module.yaml doesn't exist, don't bother returning error
		return nil, nil
	}

	runnableBytes, err := os.ReadFile(moduleFilepath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to ReadFile")
	}

	module := &tenant.Module{}
	if err := yaml.Unmarshal(runnableBytes, module); err != nil {
		return nil, errors.Wrap(err, "failed to Unmarshal")
	}

	return module, nil
}
