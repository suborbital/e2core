//nolint

package coordinator

import (
	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/suborbital/e2core/scheduler"
	"github.com/suborbital/e2core/server/coordinator/sequence"
	"github.com/suborbital/systemspec/request"
	"github.com/suborbital/systemspec/tenant"
	"github.com/suborbital/vektor/vk"
)

// nolint
type rtFunc func(scheduler.Job, *scheduler.Ctx) (interface{}, error)

// nolint
// scheduledRunner is a runner that will run a schedule on a.... schedule.
type scheduledRunner struct {
	RunFunc rtFunc
}

// nolint
func (s *scheduledRunner) Run(job scheduler.Job, ctx *scheduler.Ctx) (interface{}, error) {
	return s.RunFunc(job, ctx)
}

// nolint
func (s *scheduledRunner) OnChange(_ scheduler.ChangeEvent) error { return nil }

// nolint
func (c *Coordinator) rtFuncForSchedule(wfl tenant.Workflow) rtFunc {
	return func(job scheduler.Job, ctx *scheduler.Ctx) (interface{}, error) {
		c.log.Info("executing schedule", wfl.Name)

		req := &request.CoordinatedRequest{
			Method:  e2coreMethodSchedule,
			URL:     wfl.Name,
			ID:      uuid.New().String(),
			Body:    []byte{},
			Headers: map[string]string{},
			Params:  map[string]string{},
			State:   map[string][]byte{},
		}

		// a sequence executes the handler's steps and manages its state.
		seq, err := sequence.New(wfl.Steps, req, vk.NewCtx(c.log, nil, nil))
		if err != nil {
			c.log.Error(errors.Wrap(err, "failed to sequence.New"))
			return nil, nil
		}

		if err := seq.Execute(c.exec); err != nil {
			if runErr, isRunErr := err.(scheduler.RunErr); isRunErr {
				c.log.Error(errors.Wrapf(runErr, "workflow %s returned an error", wfl.Name))
			} else {
				c.log.Error(errors.Wrapf(err, "workflow %s failed", wfl.Name))
			}
		}

		return nil, nil
	}
}
