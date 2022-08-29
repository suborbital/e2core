package coordinator

import (
	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/suborbital/appspec/request"
	"github.com/suborbital/appspec/tenant"
	"github.com/suborbital/deltav/scheduler"
	"github.com/suborbital/deltav/server/coordinator/sequence"
	"github.com/suborbital/vektor/vk"
)

// scheduledRunner is a runner that will run a schedule on a.... schedule.
type scheduledRunner struct {
	RunFunc rtFunc
}

func (s *scheduledRunner) Run(job scheduler.Job, ctx *scheduler.Ctx) (interface{}, error) {
	return s.RunFunc(job, ctx)
}

func (s *scheduledRunner) OnChange(_ scheduler.ChangeEvent) error { return nil }

func (c *Coordinator) rtFuncForDirectiveSchedule(wfl tenant.Workflow) rtFunc {
	return func(job scheduler.Job, ctx *scheduler.Ctx) (interface{}, error) {
		c.log.Info("executing schedule", wfl.Name)

		req := &request.CoordinatedRequest{
			Method:  deltavMethodSchedule,
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
