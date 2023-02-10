// Package metrics provides a factory function that resolves to either a none, or an otel implementation
// of metrics code.
package metrics

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/metric/instrument"

	"github.com/suborbital/e2core/sat/sat/options"
)

type Metrics struct {
	FunctionExecutions       instrument.Int64Counter
	FailedFunctionExecutions instrument.Int64Counter
	FunctionTime             instrument.Int64Histogram
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
