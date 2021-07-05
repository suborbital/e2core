package rcap

import "github.com/suborbital/vektor/vlog"

// LoggerSource provides a logger to Runnables
type LoggerSource interface {
	Logger() *vlog.Logger
}

type loggerSource struct {
	log *vlog.Logger
}

// DefaultLoggerSource returns a LoggerSource that provides vlog.Default
func DefaultLoggerSource(logger *vlog.Logger) LoggerSource {
	l := &loggerSource{
		log: logger,
	}

	return l
}

// Logger returns the logger
func (l *loggerSource) Logger() *vlog.Logger {
	return l.log
}
