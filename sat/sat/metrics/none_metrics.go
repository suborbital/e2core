package metrics

import (
	"go.opentelemetry.io/otel/metric/noop"
)

func SetupNoopMetrics() Metrics {
	return Metrics{
		FunctionExecutions:       noop.Int64Counter{},
		FailedFunctionExecutions: noop.Int64Counter{},
		FunctionTime:             noop.Int64Histogram{},
	}
}
