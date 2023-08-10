// Package metrics provides a factory function that resolves to either a none, or an otel implementation
// of metrics code.
package metrics

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/metric"

	"github.com/suborbital/e2core/sat/sat/options"
)

// Meter is a global so we don't need to pass around the metrics anywhere after creating it. It follows the tracer
// pattern.
var Meter Metrics

type Metrics struct {
	FunctionExecutions       metric.Int64Counter
	FailedFunctionExecutions metric.Int64Counter
	FunctionTime             metric.Int64Histogram
	InstantiateTime          metric.Int64Histogram
}

type Timer struct {
	start time.Time
}

// ObserveMs returns the number of ms passed since NewTimer was called.
func (t Timer) ObserveMs() int64 {
	return time.Now().Sub(t.start).Milliseconds()
}

func (t Timer) ObserveMicroS() int64 {
	return time.Now().Sub(t.start).Microseconds()
}

// ObserveNs returns the number of nanoseconds passed since NewTimer was called.
func (t Timer) ObserveNs() int64 {
	return time.Now().Sub(t.start).Nanoseconds()
}

// NewTimer returns a Timer with the current time stored in it.
func NewTimer() Timer {
	return Timer{start: time.Now()}
}

func ResolveMetrics(ctx context.Context, config options.MetricsConfig, l zerolog.Logger) error {
	switch config.Type {
	case "otel":
		l.Info().Msg("setting up otel metrics")
		meter, err := setupOtelMetrics(ctx, config)
		if err != nil {
			l.Err(err).Msg("that failed sadly")
			return errors.Wrap(err, "setupOtelMetrics")
		}

		l.Info().Msg("setting global meter to the otel metrics thingy")
		Meter = meter

		return nil
	default:
		l.Info().Msg("setting up noop metrics")
		Meter = SetupNoopMetrics()

		return nil
	}
}
