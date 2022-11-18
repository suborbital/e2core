package sequence

import (
	"time"

	"github.com/pkg/errors"

	"github.com/suborbital/systemspec/tenant/executable"
)

// runGroup runs a group of functions
// this is all more complicated than it needs to be, Grav should be doing more of the work for us here.
func (seq *Sequence) ExecGroup(mods []executable.ExecutableMod) ([]FnResult, error) {
	start := time.Now()
	defer func() {
		seq.log.Debug("group executed in", time.Since(start).Milliseconds(), "ms")
	}()

	resultChan := make(chan FnResult, len(mods))

	// for now we'll use a bit of a kludgy means of running all of the group mods concurrently
	// in the future, we should send out all of the messages first, then have some new Grav
	// functionality to collect all the responses, probably using the parent ID.
	for i := range mods {
		mod := mods[i]
		seq.log.Debug("running fn", mod.FQMN, "from group")

		go func() {
			res, err := seq.ExecSingleMod(mod)
			if err != nil {
				seq.log.Error(errors.Wrap(err, "failed to runSingleFn"))
				resultChan <- FnResult{ExecErr: err.Error()}
			} else {
				resultChan <- *res
			}
		}()
	}

	results := []FnResult{}
	respCount := 0
	timeoutChan := time.After(30 * time.Second)

	for respCount < len(mods) {
		select {
		case result := <-resultChan:
			if result.ExecErr != "" {
				// if there was an error running the funciton, return that error.
				return nil, errors.New(result.ExecErr)
			}

			results = append(results, result)
		case <-timeoutChan:
			return nil, errors.New("fn group timed out")
		}

		respCount++
	}

	return results, nil
}
