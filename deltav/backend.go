package deltav

import "context"

// Backend describes something that can orchestrate DeltaV modules
type Backend interface {
	Start(ctx context.Context) error
}
