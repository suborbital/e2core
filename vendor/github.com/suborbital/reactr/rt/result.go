package rt

import (
	"encoding/json"

	"github.com/pkg/errors"
)

// Result describes a result
type Result struct {
	uuid string
	data interface{}
	err  error

	resultChan chan bool
	errChan    chan bool
}

// ResultFunc is a result callback function.
type ResultFunc func(interface{}, error)

func newResult(uuid string) *Result {
	r := &Result{
		uuid:       uuid,
		resultChan: make(chan bool, 1), // buffered, so the result can be written and related goroutines can end before Then() is called
		errChan:    make(chan bool, 1),
	}

	return r
}

// UUID returns the result/job's UUID
func (r *Result) UUID() string {
	return r.uuid
}

// Then returns the result or error from a Result
func (r *Result) Then() (interface{}, error) {
	select {
	case <-r.resultChan:
		return r.data, nil
	case <-r.errChan:
		return nil, r.err
	}
}

// ThenInt returns the result or error from a Result
func (r *Result) ThenInt() (int, error) {
	res, err := r.Then()
	if err != nil {
		return 0, err
	}

	intVal, ok := res.(int)
	if !ok {
		return 0, errors.New("failed to convert result to Int")
	}

	return intVal, nil
}

// ThenJSON unmarshals the result or returns the error from a Result
func (r *Result) ThenJSON(out interface{}) error {
	res, err := r.Then()
	if err != nil {
		return err
	}

	b, ok := res.([]byte)
	if !ok {
		return errors.New("cannot unmarshal, result is not []byte")
	}

	if err := json.Unmarshal(b, out); err != nil {
		return errors.Wrap(err, "failed to Unmarshal result")
	}

	return nil
}

// ThenDo accepts a callback function to be called asynchronously when the result completes.
func (r *Result) ThenDo(do ResultFunc) {
	go func() {
		res, err := r.Then()
		do(res, err)
	}()
}

// Discard returns immediately and discards the eventual results and thus prevents the memory from hanging around
func (r *Result) Discard() {
	go func() {
		r.Then()
	}()
}

func (r *Result) sendResult(data interface{}) {
	// if the result is another Result,
	// wait for its result and recursively send it
	// or if the result is a group, wait on the
	// group and propogate the error if any
	if res, ok := data.(*Result); ok {
		go func() {
			if newResult, err := res.Then(); err != nil {
				r.sendErr(err)
			} else {
				r.sendResult(newResult)
			}
		}()

		return
	} else if grp, ok := data.(*Group); ok {
		go func() {
			if err := grp.Wait(); err != nil {
				r.sendErr(err)
			} else {
				r.sendResult(nil)
			}
		}()

		return
	}

	r.data = data
	r.resultChan <- true
}

func (r *Result) sendErr(err error) {
	r.err = err
	r.errChan <- true
}
