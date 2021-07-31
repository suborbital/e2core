package coordinator

import (
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/suborbital/atmo/directive"
	"github.com/suborbital/reactr/request"
	"github.com/suborbital/reactr/rt"
	"github.com/suborbital/vektor/vk"
)

// scheduledRunner is a runner that will run a schedule on a.... schedule
type scheduledRunner struct {
	RunFunc rtFunc
}

func (s *scheduledRunner) Run(job rt.Job, ctx *rt.Ctx) (interface{}, error) {
	return s.RunFunc(job, ctx)
}

func (s *scheduledRunner) OnChange(_ rt.ChangeEvent) error { return nil }

func (c *Coordinator) rtFuncForDirectiveSchedule(sched directive.Schedule) rtFunc {
	return func(job rt.Job, ctx *rt.Ctx) (interface{}, error) {
		c.log.Info("executing schedule", sched.Name)

		// read the "initial" state from the Directive
		state := map[string][]byte{}
		for k, v := range sched.State {
			state[k] = []byte(v)
		}

		req := &request.CoordinatedRequest{
			Method:  atmoMethodSchedule,
			URL:     sched.Name,
			ID:      uuid.New().String(),
			Body:    []byte{},
			Headers: map[string]string{},
			Params:  map[string]string{},
			State:   state,
		}

		// a sequence executes the handler's steps and manages its state
		seq := newSequence(sched.Steps, c.grav.Connect, vk.NewCtx(c.log, nil, nil))

		if seqState, err := seq.exec(req); err != nil {
			if errors.Is(err, ErrSequenceRunErr) && seqState.err != nil {
				c.log.Error(errors.Wrapf(seqState.err, "schedule %s returned an error", sched.Name))
			} else {
				c.log.Error(errors.Wrapf(err, "schedule %s failed", sched.Name))
			}
		}

		return nil, nil
	}
}
