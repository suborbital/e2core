package sat

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/suborbital/e2core/foundation/scheduler"
	"github.com/suborbital/e2core/foundation/tracing"
	"github.com/suborbital/systemspec/request"
)

// Exec takes input bytes, executes the loaded module, and returns the result
func (s *Sat) Exec(ctx context.Context, input []byte) (*request.CoordinatedResponse, error) {
	ctx, span := tracing.Tracer.Start(ctx, "sat.Exec")
	defer span.End()

	// construct a fake HTTP request from the input
	req := &request.CoordinatedRequest{
		Method:      http.MethodPost,
		URL:         "/",
		ID:          uuid.New().String(),
		Body:        input,
		Headers:     map[string]string{},
		RespHeaders: map[string]string{},
		Params:      map[string]string{},
		State:       map[string][]byte{},
	}

	result, err := s.engine.Do(scheduler.NewJob(s.config.JobType, req).WithContext(ctx)).Then()
	if err != nil {
		return nil, errors.Wrap(err, "failed to exec")
	}

	resp, ok := result.(*request.CoordinatedResponse)
	if !ok {
		return nil, errors.New("response is not a coordinated response")
	}

	return resp, nil
}
