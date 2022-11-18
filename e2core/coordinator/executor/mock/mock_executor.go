package mock

import (
	"fmt"

	"github.com/suborbital/e2core/e2core/coordinator/executor"
	"github.com/suborbital/e2core/foundation/bus/bus"
	"github.com/suborbital/e2core/foundation/scheduler"
	"github.com/suborbital/systemspec/fqmn"
	"github.com/suborbital/systemspec/request"
	"github.com/suborbital/systemspec/tenant/executable"
	"github.com/suborbital/vektor/vk"
)

type JobFunc func(interface{}, *vk.Ctx) (interface{}, error)

type Executor struct {
	Jobs map[string]JobFunc
}

// Do executes the mock job
func (m *Executor) Do(jobType string, req *request.CoordinatedRequest, ctx *vk.Ctx, cb bus.MsgFunc) (interface{}, error) {
	jobFunc, exists := m.Jobs[jobType]
	if !exists {
		return nil, executor.ErrCannotHandle
	}

	return jobFunc(req, ctx)
}

// DesiredStepState was copied from Sat
func (m *Executor) DesiredStepState(step executable.Executable, req *request.CoordinatedRequest) (map[string][]byte, error) {
	if len(step.With) == 0 {
		return nil, executor.ErrDesiredStateNotGenerated
	}

	desiredState := map[string][]byte{}
	aliased := map[string]bool{}

	// first go through the 'with' clause and load all of the appropriate aliased values.
	for alias, key := range step.With {
		val, exists := req.State[key]
		if !exists {
			// if the literal key is not in state,
			// iterate through all the state keys and
			// parse them as FQMNs, and match with any
			// that have matching names.

			found := false

			for stateKey := range req.State {
				stateFQMN, err := fqmn.Parse(stateKey)
				if err != nil {
					// if the state key isn't an FQMN, that's fine, move along
					continue
				}

				if stateFQMN.Name == key {
					found = true

					val = req.State[stateKey]

					desiredState[alias] = val
					aliased[stateKey] = true

					break
				}
			}

			if !found {
				return nil, fmt.Errorf("failed to build desired state, %s does not exists in handler state", key)
			}
		} else {
			desiredState[alias] = val
			aliased[key] = true
		}

	}

	// next, go through the rest of the original state and load the non-aliased values.
	for key, val := range req.State {
		_, skip := aliased[key]
		if !skip {
			desiredState[key] = val
		}
	}

	return desiredState, nil
}

func (m *Executor) SetSchedule(sched scheduler.Schedule) error {
	return nil
}

func (m *Executor) Metrics() (*scheduler.ScalerMetrics, error) {
	return nil, nil
}
