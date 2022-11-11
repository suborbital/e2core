package sat

import (
	"go.opentelemetry.io/otel/metric/instrument/syncint64"
)

type Metrics struct {
	FunctionExecutions       syncint64.Counter
	FailedFunctionExecutions syncint64.Counter
	FunctionTime             syncint64.Histogram
}
