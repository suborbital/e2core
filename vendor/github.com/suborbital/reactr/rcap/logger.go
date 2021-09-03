package rcap

import "github.com/suborbital/vektor/vlog"

// LoggerConfig is configuration for the logger capability
type LoggerConfig struct {
	Enabled bool         `json:"enabled" yaml:"enabled"`
	Logger  *vlog.Logger `json:"-" yaml:"-"`
}

// LoggerCapability provides a logger to Runnables
type LoggerCapability interface {
	Log(level int32, msg string, scope interface{})
}

type loggerSource struct {
	config LoggerConfig
	log    *vlog.Logger
}

// DefaultLoggerSource returns a LoggerSource that provides vlog.Default
func DefaultLoggerSource(config LoggerConfig) LoggerCapability {
	l := &loggerSource{
		config: config,
		log:    config.Logger,
	}

	return l
}

// Log level int32, msg stringreturns the logger
func (l *loggerSource) Log(level int32, msg string, scope interface{}) {
	if !l.config.Enabled {
		return
	}

	scoped := l.log.CreateScoped(scope)

	switch level {
	case 1:
		scoped.ErrorString(msg)
	case 2:
		scoped.Warn(msg)
	case 4:
		scoped.Debug(msg)
	default:
		scoped.Info(msg)
	}
}
