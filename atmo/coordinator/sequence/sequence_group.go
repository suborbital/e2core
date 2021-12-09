package sequence

import (
	"time"

	"github.com/pkg/errors"
	"github.com/suborbital/atmo/directive/executable"
	"github.com/suborbital/reactr/request"
)

// runGroup runs a group of functions
// this is all more complicated than it needs to be, Grav should be doing more of the work for us here
func (seq *Sequence) ExecGroup(fns []executable.CallableFn, req *request.CoordinatedRequest) ([]FnResult, error) {
	start := time.Now()
	defer func() {
		seq.log.Debug("group executed in", time.Since(start).Milliseconds(), "ms")
	}()

	resultChan := make(chan FnResult, len(fns))

	// for now we'll use a bit of a kludgy means of running all of the group fns concurrently
	// in the future, we should send out all of the messages first, then have some new Grav
	// functionality to collect all the responses, probably using the parent ID.
	for i := range fns {
		fn := fns[i]
		seq.log.Debug("running fn", fn.Fn, "from group")

		go func() {
			res, err := seq.ExecSingleFn(fn, req)
			if err != nil {
				seq.log.Error(errors.Wrap(err, "failed to runSingleFn"))
				resultChan <- FnResult{ExecErr: err}
			} else {
				resultChan <- *res
			}
		}()
	}

	results := []FnResult{}
	respCount := 0
	timeoutChan := time.After(30 * time.Second)

	for respCount < len(fns) {
		select {
		case result := <-resultChan:
			if result.ExecErr != nil {
				// if there was an error running the funciton, return that error
				return nil, result.ExecErr
			}

			results = append(results, result)
		case <-timeoutChan:
			return nil, errors.New("fn group timed out")
		}

		respCount++
	}

	return results, nil
}
