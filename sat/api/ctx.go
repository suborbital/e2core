package api

import (
	"context"

	"github.com/suborbital/appspec/request"
)

type ctxKey int

const requestKey = ctxKey(0)

// ContextWithRequest returns the provided context with a request object added as a value
func ContextWithRequest(ctx context.Context, req *request.CoordinatedRequest) context.Context {
	return context.WithValue(ctx, requestKey, req)
}

// RequestFromContext returns the stored request from a given context, if any
func RequestFromContext(ctx context.Context) *request.CoordinatedRequest {
	req := ctx.Value(requestKey)

	if req != nil {
		if coordReq, ok := req.(*request.CoordinatedRequest); ok {
			return coordReq
		}
	}

	return nil
}
