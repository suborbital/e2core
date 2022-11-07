package metrics

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric/instrument"
)

func SetupNoopMetrics() Metrics {
	return Metrics{
		FunctionExecutions:       noopCounter{},
		FailedFunctionExecutions: noopCounter{},
		FunctionTime:             noopHistogram{},
	}
}

// noopCounter implements the metrics.noopCounter interface and the instrument.Synchronous interface. It does nothing, and can
// be used anywhere we need a metrics.noopCounter implementation.
type noopCounter struct {
	instrument.Synchronous
}

// Add implements the metrics.noopCounter interface method.
func (f noopCounter) Add(_ context.Context, _ int64, _ ...attribute.KeyValue) {
	// no op, do nothing
}

// noopHistogram implements the metrics.noopHistogram interface and the instrument.Synchronous interface. It does nothing, and
// can be used anywhere we need a metrics.noopHistogram implementation.
type noopHistogram struct {
	instrument.Synchronous
}

// Record implements the metrics.noopHistogram interface method.
func (h noopHistogram) Record(_ context.Context, _ int64, _ ...attribute.KeyValue) {
	// no op, do nothing
}
