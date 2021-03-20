package coordinator

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/suborbital/atmo/directive"
	"github.com/suborbital/reactr/request"
)

// runForEach runs a forEach on an array input
// this is all more complicated than it needs to be, Grav should be doing more of the work for us here
func (seq *sequence) runForEach(forEach *directive.ForEach, req request.CoordinatedRequest) (*fnResult, error) {
	start := time.Now()
	defer func() {
		seq.log.Debug("forEach executed in", time.Since(start).Milliseconds(), "ms")
	}()

	val, ok := req.State[forEach.In]
	if !ok {
		return nil, fmt.Errorf("forEach failed, %s is not present in state", forEach.In)
	}

	// turn the array into an array of bytes for each element
	arrayVals, err := arrayFromStateVal(val)
	if err != nil {
		return nil, errors.Wrap(err, "failed to arrayFromStateVal")
	}

	// prepare to loop over all the array values
	fn := directive.CallableFn{Fn: forEach.Fn, OnErr: forEach.OnErr, As: forEach.As}
	resultChan := make(chan fnResult, len(arrayVals))

	// run the fn for each element in the array
	// this is even kludgier than runGroup, but alas...
	for i := range arrayVals {
		val := arrayVals[i]

		seq.log.Debug("running fn", fn.Fn, "from forEach on element", i)
		// add the iteration value to a spectial state field
		req.State["__elem"] = val

		reqJSON, err := req.ToJSON()
		if err != nil {
			return nil, errors.Wrap(err, "failed to forEachReq.ToJSON")
		}

		go func() {
			res, err := seq.runSingleFn(fn, reqJSON)
			if err != nil {
				seq.log.Error(errors.Wrap(err, "failed to runSingleFn"))
				resultChan <- fnResult{err: err}
			} else {
				resultChan <- *res
			}
		}()
	}

	resultElements := [][]byte{}
	respCount := 0
	timeoutChan := time.After(30 * time.Second)

	for respCount < len(arrayVals) {
		select {
		case result := <-resultChan:
			if result.err != nil {
				// if there was an error running the funciton, return that error
				return nil, result.err
			} else if result.runErr != nil {
				// if there was an error returned from the fn, propogate that
				// and let the sequence logic handle it
				return &result, nil
			}

			resultElements = append(resultElements, result.response.Output)
		case <-timeoutChan:
			return nil, errors.New("fn group timed out")
		}

		respCount++
	}

	// re-build the results array
	resultJSON, err := stateValFromArray(resultElements)
	if err != nil {
		return nil, errors.Wrap(err, "failed to stateValFromArray")
	}

	// calculate the FQFN again since we lost it when collecting results
	fqfn, err := seq.fqfn(fn.Fn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to FQFN for fn %s", fn.Fn)
	}

	result := &fnResult{
		fqfn:     fqfn,
		key:      forEach.As,
		response: &request.CoordinatedResponse{Output: resultJSON},
	}

	return result, nil
}

func arrayFromStateVal(stateVal []byte) ([][]byte, error) {
	jsonArray := []interface{}{}

	if err := json.Unmarshal(stateVal, &jsonArray); err != nil {
		return nil, errors.Wrap(err, "failed to Unmarshal into jsonArray")
	}

	results := [][]byte{}

	for i, elem := range jsonArray {
		jsonBytes, err := json.Marshal(elem)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to Marshal element at position %d", i)
		}

		results = append(results, jsonBytes)
	}

	return results, nil
}

func stateValFromArray(array [][]byte) ([]byte, error) {
	jsonArray := []interface{}{}

	for i, bytes := range array {
		elem := map[string]interface{}{}
		if err := json.Unmarshal(bytes, &elem); err != nil {
			return nil, errors.Wrapf(err, "failed to Marshal element at position %d", i)
		}

		jsonArray = append(jsonArray, elem)
	}

	jsonBytes, err := json.Marshal(jsonArray)
	if err != nil {
		return nil, errors.Wrap(err, "failed to Marshal results array")
	}

	return jsonBytes, nil
}
