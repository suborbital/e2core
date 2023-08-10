package metrics

import (
	"context"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/embedded"
)

func SetupNoopMetrics() Metrics {
	return Metrics{
		FunctionExecutions:       noopCounter{},
		FailedFunctionExecutions: noopCounter{},
		FunctionTime:             noopHistogram{},
		InstantiateTime:          noopHistogram{},
	}
}

var _ metric.Int64Counter = noopCounter{}

// noopCounter implements the metrics.noopCounter interface and the instrument.Synchronous interface. It does nothing, and can
// be used anywhere we need a metrics.noopCounter implementation.
type noopCounter struct {
	embedded.Int64Counter
}

// Add implements the metric.Int64Counter interface method.
func (f noopCounter) Add(_ context.Context, _ int64, _ ...metric.AddOption) {
	// do nothing
}

var _ metric.Int64Histogram = noopHistogram{}

// noopHistogram implements the metrics.noopHistogram interface and the instrument.Synchronous interface. It does nothing, and
// can be used anywhere we need a metrics.noopHistogram implementation.
type noopHistogram struct {
	embedded.Int64Histogram
}

func (h noopHistogram) Record(_ context.Context, _ int64, _ ...metric.RecordOption) {
	// do nothing, noop
}
