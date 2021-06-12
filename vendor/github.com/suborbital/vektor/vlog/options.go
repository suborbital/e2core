package vlog

import (
	"context"
	"fmt"
	"strings"

	"github.com/sethvargo/go-envconfig"
)

const defaultEnvPrefix = "VLOG"

// LogLevelTrace and others represent log levels
const (
	LogLevelTrace = "trace" // 5
	LogLevelDebug = "debug" // 4
	LogLevelInfo  = "info"  // 3
	LogLevelWarn  = "warn"  // 2
	LogLevelError = "error" // 1
)

var levelStringMap = map[string]int{
	LogLevelTrace: 5,
	LogLevelDebug: 4,
	LogLevelInfo:  3,
	LogLevelWarn:  2,
	LogLevelError: 1,
}

// Options represents the options for a VLogger
type Options struct {
	Level       int         `env:"-"`
	LevelString string      `env:"_LOG_LEVEL"`
	Filepath    string      `env:"_LOG_FILE"`
	LogPrefix   string      `env:"_LOG_PREFIX"`
	EnvPrefix   string      `env:"-"`
	AppMeta     interface{} `env:"-"`
	PreLogHook  LogHookFunc `env:"-"`
}

type LogHookFunc func([]byte)

// OptionsModifier is a options modifier function
type OptionsModifier func(*Options)

func newOptions(mods ...OptionsModifier) *Options {
	opts := defaultOptions()

	for _, mod := range mods {
		mod(opts)
	}

	envPrefix := defaultEnvPrefix
	if opts.EnvPrefix != "" {
		envPrefix = opts.EnvPrefix
	}

	opts.finalize(envPrefix)

	return opts
}

// Level sets the logging level to one of error, warn, info, debug, or trace
func Level(level string) OptionsModifier {
	return func(opt *Options) {
		opt.Level = logLevelValFromString(level)
	}
}

// ToFile sets the logger to open the file specified and write logs to it
func ToFile(filepath string) OptionsModifier {
	return func(opt *Options) {
		opt.Filepath = filepath
	}
}

// LogPrefix sets a prefix on all of the log messages
func LogPrefix(logPrefix string) OptionsModifier {
	return func(opt *Options) {
		opt.LogPrefix = logPrefix
	}
}

// EnvPrefix sets a prefix for evaluating logger settings from env
func EnvPrefix(envPrefix string) OptionsModifier {
	return func(opt *Options) {
		opt.EnvPrefix = envPrefix
	}
}

// AppMeta sets the AppMeta object to be included with structured logs
func AppMeta(meta interface{}) OptionsModifier {
	return func(opt *Options) {
		opt.AppMeta = meta
	}
}

// PreLogHook sets a function to be run before each logged value
func PreLogHook(hook LogHookFunc) OptionsModifier {
	return func(opt *Options) {
		opt.PreLogHook = hook
	}
}

func defaultOptions() *Options {
	o := &Options{
		Level:       logLevelValFromString(LogLevelInfo),
		LevelString: "",
		Filepath:    "",
		LogPrefix:   "",
		EnvPrefix:   "",
		AppMeta:     nil,
	}

	return o
}

// finalize "locks in" the options by overriding any existing options with the version from the environment, and setting the default logger if needed
func (o *Options) finalize(envPrefix string) {
	envOpts := Options{}
	if err := envconfig.ProcessWith(context.Background(), &envOpts, envconfig.PrefixLookuper(envPrefix, envconfig.OsLookuper())); err != nil {
		fmt.Printf("[vlog] failed to ProcessWith environment config:" + err.Error())
		return
	}

	o.replaceFieldsIfNeeded(&envOpts)
}

func (o *Options) replaceFieldsIfNeeded(replacement *Options) {
	if replacement.LevelString != "" {
		o.Level = logLevelValFromString(replacement.LevelString)
	}

	if replacement.Filepath != "" {
		o.Filepath = replacement.Filepath
	}

	if replacement.LogPrefix != "" {
		o.LogPrefix = replacement.LogPrefix
	}
}

func logLevelValFromString(level string) int {
	if level, ok := levelStringMap[strings.ToLower(level)]; ok {
		return level
	}

	return 3
}
