package backend

import "context"

// Backend describes something that can orchestrate E2Core modules
type Backend interface {
	Start(ctx context.Context) error
}
