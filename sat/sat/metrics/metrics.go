// Package metrics provides a factory function that resolves to either a none, or an otel implementation
// of metrics code.
package metrics

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/metric/instrument/syncint64"

	"github.com/suborbital/e2core/sat/sat/options"
)

type Metrics struct {
	FunctionExecutions       syncint64.Counter
	FailedFunctionExecutions syncint64.Counter
	FunctionTime             syncint64.Histogram
}

type Timer struct {
	start time.Time
}

func (t Timer) Observe() int64 {
	return time.Now().Sub(t.start).Milliseconds()
}

// NewTimer returns a Timer with the current time stored in it.
func NewTimer() Timer {
	return Timer{start: time.Now()}
}

func ResolveMetrics(ctx context.Context, config options.MetricsConfig) (Metrics, error) {
	switch config.Type {
	case "otel":
		return setupOtelMetrics(ctx, config)
	default:
		return SetupNoopMetrics(), nil
	}
}
