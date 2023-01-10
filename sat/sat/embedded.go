package sat

import (
	"net/http"

	"github.com/pkg/errors"

	"github.com/suborbital/e2core/foundation/scheduler"
	"github.com/suborbital/systemspec/request"
	"github.com/suborbital/vektor/vk"
)

// Exec takes input bytes, executes the loaded module, and returns the result
func (s *Sat) Exec(input []byte) (*request.CoordinatedResponse, error) {
	ctx := vk.NewCtx(s.config.Logger, nil, nil)

	// construct a fake HTTP request from the input
	req := &request.CoordinatedRequest{
		Method:      http.MethodPost,
		URL:         "/",
		ID:          ctx.RequestID(),
		Body:        input,
		Headers:     map[string]string{},
		RespHeaders: map[string]string{},
		Params:      map[string]string{},
		State:       map[string][]byte{},
	}

	result, err := s.engine.Do(scheduler.NewJob(s.jobName, req)).Then()
	if err != nil {
		return nil, errors.Wrap(err, "failed to exec")
	}

	resp := result.(*request.CoordinatedResponse)

	return resp, nil
}
