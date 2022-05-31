package coordinator

import (
	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/suborbital/vektor/vk"
	"github.com/suborbital/velocity/directive"
	"github.com/suborbital/velocity/scheduler"
	"github.com/suborbital/velocity/server/coordinator/sequence"
	"github.com/suborbital/velocity/server/request"
)

// scheduledRunner is a runner that will run a schedule on a.... schedule.
type scheduledRunner struct {
	RunFunc rtFunc
}

func (s *scheduledRunner) Run(job scheduler.Job, ctx *scheduler.Ctx) (interface{}, error) {
	return s.RunFunc(job, ctx)
}

func (s *scheduledRunner) OnChange(_ scheduler.ChangeEvent) error { return nil }

func (c *Coordinator) rtFuncForDirectiveSchedule(sched directive.Schedule) rtFunc {
	return func(job scheduler.Job, ctx *scheduler.Ctx) (interface{}, error) {
		c.log.Info("executing schedule", sched.Name)

		// read the "initial" state from the Directive.
		state := map[string][]byte{}
		for k, v := range sched.State {
			state[k] = []byte(v)
		}

		req := &request.CoordinatedRequest{
			Method:  velocityMethodSchedule,
			URL:     sched.Name,
			ID:      uuid.New().String(),
			Body:    []byte{},
			Headers: map[string]string{},
			Params:  map[string]string{},
			State:   state,
		}

		// a sequence executes the handler's steps and manages its state.
		seq, err := sequence.New(sched.Steps, req, vk.NewCtx(c.log, nil, nil))
		if err != nil {
			c.log.Error(errors.Wrap(err, "failed to sequence.New"))
			return nil, nil
		}

		if err := seq.Execute(c.exec); err != nil {
			if runErr, isRunErr := err.(scheduler.RunErr); isRunErr {
				c.log.Error(errors.Wrapf(runErr, "schedule %s returned an error", sched.Name))
			} else {
				c.log.Error(errors.Wrapf(err, "schedule %s failed", sched.Name))
			}
		}

		return nil, nil
	}
}
