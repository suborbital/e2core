package vlog

import (
	"fmt"
)

type defaultProducer struct{}

// ErrorString prints a string as an error
func (d *defaultProducer) ErrorString(msgs ...interface{}) string {
	return fmt.Sprintf("(E) %s", redactAndJoinInterfaces(msgs...))
}

// Error prints a string as an error
func (d *defaultProducer) Error(err error) string {
	return fmt.Sprintf("(E) %s", err.Error())
}

// Warn prints a string as an warning
func (d *defaultProducer) Warn(msgs ...interface{}) string {
	return fmt.Sprintf("(W) %s", redactAndJoinInterfaces(msgs...))
}

// Info prints a string as an info message
func (d *defaultProducer) Info(msgs ...interface{}) string {
	return fmt.Sprintf("(I) %s", redactAndJoinInterfaces(msgs...))
}

// Debug prints a string as debug output
func (d *defaultProducer) Debug(msgs ...interface{}) string {
	return fmt.Sprintf("(D) %s", redactAndJoinInterfaces(msgs...))
}

// Trace prints a function name and returns a function to be deferred, logging the completion of a function
func (d *defaultProducer) Trace(fnName string) (string, func() string) {
	traceFunc := func() string {
		return (fmt.Sprintf("(T) %s completed", fnName))
	}

	return (fmt.Sprintf("(T) %s", fnName)), traceFunc
}

func redactAndJoinInterfaces(msgs ...interface{}) string {
	msg := ""

	for _, m := range msgs {
		str, ok := m.(string)
		if ok {
			msg += fmt.Sprintf(" %s", str)
		} else {
			msg += fmt.Sprintf(" [redacted %T]", m)
		}
	}

	// get rid of that first space
	return msg[1:]
}
