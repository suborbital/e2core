package sat

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"gopkg.in/yaml.v3"

	satOptions "github.com/suborbital/e2core/sat/sat/options"
	"github.com/suborbital/systemspec/capabilities"
	"github.com/suborbital/systemspec/fqmn"
	"github.com/suborbital/systemspec/system"
	"github.com/suborbital/systemspec/system/client"
	"github.com/suborbital/systemspec/tenant"
)

type Config struct {
	ModuleArg       string
	JobType         string
	PrettyName      string
	Module          *tenant.Module
	Tenant          string
	CapConfig       capabilities.CapabilityConfig
	Connections     []tenant.Connection
	Port            int
	ControlPlaneUrl string
	EnvToken        string
	ProcUUID        string
	TracerConfig    satOptions.TracerConfig
	MetricsConfig   satOptions.MetricsConfig
}

func ConfigFromArgs(l zerolog.Logger, opts satOptions.Options) (*Config, error) {
	flag.Parse()
	args := flag.Args()

	if len(args) < 1 {
		return nil, errors.New("missing argument: module (path, URL or FQMN)")
	}

	moduleArg := args[0]

	return ConfigFromModuleArg(l, opts, moduleArg)
}

func ConfigFromModuleArg(logger zerolog.Logger, opts satOptions.Options, moduleArg string) (*Config, error) {
	var module *tenant.Module
	var FQMN fqmn.FQMN
	var err error

	// first, determine if we need to connect to a control plane
	controlPlane := ""
	useControlPlane := false
	if opts.ControlPlane != nil {
		controlPlane = opts.ControlPlane.Address
		useControlPlane = true
	}

	appClient := client.NewHTTPSource(controlPlane, NewAuthToken(opts.EnvToken))
	caps := capabilities.DefaultCapabilityConfig()

	if useControlPlane {
		if err = appClient.Start(); err != nil {
			return nil, errors.Wrap(err, "failed to systemSource.Start")
		}
	}

	// next, handle the module arg being a URL, an FQMN, or a path on disk
	if isURL(moduleArg) {
		logger.Debug().Msg("fetching module from URL")
		tmpFile, err := downloadFromURL(moduleArg)
		if err != nil {
			return nil, errors.Wrap(err, "failed to downloadFromURL")
		}

		moduleArg = tmpFile
	} else if FQMN, err = fqmn.Parse(moduleArg); err == nil {
		if useControlPlane {
			logger.Debug().Msg("fetching module from control plane")

			cpModule, err := appClient.GetModule(moduleArg)
			if err != nil {
				return nil, errors.Wrap(err, "failed to GetModule")
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
		diskModule, err := findModuleDotYaml(moduleArg)
		if err != nil {
			return nil, errors.Wrap(err, "failed to findModuleDotYaml")
		}

		if diskModule != nil {
			if opts.Ident != nil && opts.Version != nil {
				FQMN, err := fqmn.FromParts(opts.Ident.Data, module.Namespace, module.Name, opts.Version.Data)
				if err != nil {
					return nil, errors.Wrap(err, "failed to fqmn.FromParts")
				}

				module.FQMN = FQMN
			}
		}

		module = diskModule
	}

	// set some defaults in the case we're not running in an application
	portInt, _ := strconv.Atoi(string(opts.Port))
	jobType := strings.TrimSuffix(filepath.Base(moduleArg), ".wasm")
	prettyName := jobType

	// modify configuration if we ARE running as part of an application
	if module != nil && module.FQMN != "" {
		jobType = module.FQMN

		prettyName = fmt.Sprintf("%s-%s", jobType, opts.ProcUUID[:6])

		logger = logger.With().
			Str("app", prettyName).
			Str("jobType", jobType).
			Str("tenant", FQMN.Tenant).
			Logger()

		logger.Debug().Msg("configuring")
		logger.Debug().Msg("joining tenant")
	} else {
		logger.Debug().Str("jobType", jobType).Msg("configuring")
	}

	conns := make([]tenant.Connection, 0)
	if opts.Connections != "" {
		if err := json.Unmarshal([]byte(opts.Connections), &conns); err != nil {
			return nil, errors.Wrap(err, "failed to Unmarshal connections JSON")
		}
	}

	// finally, put it all together
	c := &Config{
		ModuleArg:       moduleArg,
		JobType:         jobType,
		PrettyName:      prettyName,
		Module:          module,
		Tenant:          FQMN.Tenant,
		CapConfig:       caps,
		Connections:     conns,
		Port:            portInt,
		ControlPlaneUrl: controlPlane,
		TracerConfig:    opts.TracerConfig,
		MetricsConfig:   opts.MetricsConfig,
		ProcUUID:        string(opts.ProcUUID),
	}

	return c, nil
}

func findModuleDotYaml(moduleArg string) (*tenant.Module, error) {
	filename := filepath.Base(moduleArg)
	moduleFilepath := strings.Replace(moduleArg, filename, ".module.yml", -1)

	if _, err := os.Stat(moduleFilepath); err != nil {
		// .module.yaml doesn't exist, don't bother returning error
		return nil, nil
	}

	moduleBytes, err := os.ReadFile(moduleFilepath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to ReadFile")
	}

	module := &tenant.Module{}
	if err := yaml.Unmarshal(moduleBytes, module); err != nil {
		return nil, errors.Wrap(err, "failed to Unmarshal")
	}

	return module, nil
}
