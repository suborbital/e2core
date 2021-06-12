package vlog

import (
	"encoding/json"
	"io"
	"os"
	"sync"
	"time"
)

// Producer represents an object that is considered a producer of messages
type Producer interface {
	ErrorString(...interface{}) string    // Logs an error string
	Error(error) string                   // Logs an error obj
	Warn(...interface{}) string           // Logs a warning
	Info(...interface{}) string           // Logs information
	Debug(...interface{}) string          // Logs debug information
	Trace(string) (string, func() string) // Logs a function name and returns a function to be deferred, indicating the end of the function
}

// Logger is the main logger object, responsible for taking input from the
// producer and managing scoped loggers
type Logger struct {
	producer Producer
	scope    interface{}
	opts     *Options
	output   io.Writer
	lock     *sync.Mutex
}

// SafeStringer allows a struct to produse a "safe" string representation for logging
// the intention is avoiding accidentally including sensitive information in struct fields.
type SafeStringer interface {
	SafeString() string
}

// Default returns a Logger using the default producer
func Default(opts ...OptionsModifier) *Logger {
	prod := &defaultProducer{}

	return New(prod, opts...)
}

// New returns a Logger with the provided producer and options
func New(producer Producer, opts ...OptionsModifier) *Logger {
	options := newOptions(opts...)

	v := &Logger{
		producer: producer,
		scope:    nil,
		opts:     options,
		lock:     &sync.Mutex{},
	}

	output, err := outputForOptions(options)
	if err != nil {
		v.output = os.Stdout
		os.Stderr.Write([]byte("[vlog] failed to set output: " + err.Error() + "\n"))
	} else {
		v.output = output
	}

	return v
}

// CreateScoped creates a duplicate logger which has a particular scope
func (v *Logger) CreateScoped(scope interface{}) *Logger {
	sl := &Logger{
		producer: v.producer,
		scope:    scope,
		opts:     v.opts,
		output:   v.output,
		lock:     v.lock,
	}

	return sl
}

// ErrorString logs a string as an error
func (v *Logger) ErrorString(msgs ...interface{}) {
	msg := v.producer.ErrorString(msgs...)

	v.log(msg, v.scope, 1)
}

// Error logs an error as an error
func (v *Logger) Error(err error) {
	msg := v.producer.Error(err)

	v.log(msg, v.scope, 1)
}

// Warn logs a string as an warning
func (v *Logger) Warn(msgs ...interface{}) {
	msg := v.producer.Warn(msgs...)

	v.log(msg, v.scope, 2)
}

// Info logs a string as an info message
func (v *Logger) Info(msgs ...interface{}) {
	msg := v.producer.Info(msgs...)

	v.log(msg, v.scope, 3)
}

// Debug logs a string as debug output
func (v *Logger) Debug(msgs ...interface{}) {
	msg := v.producer.Debug(msgs...)

	v.log(msg, v.scope, 4)
}

// Trace logs a function name and returns a function to be deferred, logging the completion of a function
func (v *Logger) Trace(fnName string) func() {
	msg, traceFunc := v.producer.Trace(fnName)

	v.log(msg, v.scope, 5)

	return func() {
		msg := traceFunc()

		v.log(msg, v.scope, 5)
	}
}

func (v *Logger) log(message string, scope interface{}, level int) {
	if level > v.opts.Level {
		return
	}

	if v.opts.LogPrefix != "" {
		message = v.opts.LogPrefix + " " + message
	}

	// send the raw message to the console
	if v.output != os.Stdout {
		// acquire a lock as the output may be a file
		v.lock.Lock()
		defer v.lock.Unlock()

		// throwing away the error here since there's nothing much we can do
		os.Stdout.Write([]byte(message))
		os.Stdout.Write([]byte("\n"))
	}

	structured := structuredLog{
		LogMessage: message,
		Timestamp:  time.Now(),
		Level:      level,
		AppMeta:    v.opts.AppMeta,
		ScopeMeta:  scope,
	}

	structuredJSON, err := json.Marshal(structured)
	if err != nil {
		os.Stderr.Write([]byte("[vlog] failed to marshal structured log"))
	}

	if v.opts.PreLogHook != nil {
		v.opts.PreLogHook(structuredJSON)
	}

	_, err = v.output.Write(structuredJSON)
	if err != nil {
		os.Stderr.Write([]byte("[vlog] failed to write to configured output: " + err.Error() + "\n"))
	} else {
		v.output.Write([]byte("\n"))
	}

}

func outputForOptions(opts *Options) (io.Writer, error) {
	var output io.Writer

	if opts.Filepath != "" {
		file, err := os.OpenFile(opts.Filepath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, os.ModePerm)
		if err != nil {
			return nil, err
		}

		output = file
	} else {
		output = os.Stdout
	}

	return output, nil
}
