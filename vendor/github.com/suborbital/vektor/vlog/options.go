package vlog

// LogLevelTrace and others represent log levels
const (
	LogLevelTrace = "trace" // 5
	LogLevelDebug = "debug" // 4
	LogLevelInfo  = "info"  // 3
	LogLevelWarn  = "warn"  // 2
	LogLevelError = "error" // 1
)

// Options represents the options for a VLogger
type Options struct {
	level    int
	filepath string
	prefix   string
	appMeta  interface{}
}

// OptionsModifier is a options modifier function
type OptionsModifier func(Options) Options

func newOptions(mods ...OptionsModifier) Options {
	opts := defaultOptions()

	for _, mod := range mods {
		opts = mod(opts)
	}

	return opts
}

// Level sets the logging level to one of error, warn, info, debug, or trace
func Level(level string) OptionsModifier {
	return func(opt Options) Options {
		opt.level = logLevelValFromString(level)

		return opt
	}
}

// ToFile sets the logger to open the file specified and write logs to it
func ToFile(filepath string) OptionsModifier {
	return func(opt Options) Options {
		opt.filepath = filepath

		return opt
	}
}

// Prefix sets a prefix on all of the log messages
func Prefix(prefix string) OptionsModifier {
	return func(opt Options) Options {
		opt.prefix = prefix

		return opt
	}
}

// AppMeta sets the AppMeta object to be included with structured logs
func AppMeta(meta interface{}) OptionsModifier {
	return func(opt Options) Options {
		opt.appMeta = meta

		return opt
	}
}

func defaultOptions() Options {
	o := Options{
		level:    logLevelValFromString(LogLevelInfo),
		filepath: "",
		prefix:   "",
		appMeta:  nil,
	}

	return o
}

func logLevelValFromString(level string) int {
	switch level {
	case LogLevelTrace:
		return 5
	case LogLevelDebug:
		return 4
	case LogLevelInfo:
		return 3
	case LogLevelWarn:
		return 2
	case LogLevelError:
		return 1
	}

	return 3
}
