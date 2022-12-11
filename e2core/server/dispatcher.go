package server

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/suborbital/e2core/e2core/sequence"
	"github.com/suborbital/e2core/foundation/bus/bus"
	"github.com/suborbital/vektor/vlog"
)

const (
	MsgTypeSuborbitalResult = "suborbital.result"
)

var (
	ErrDesiredStateNotGenerated = errors.New("desired state was not generated")
	ErrDispatchTimeout          = errors.New("dispatched execution did not complete before the timeout")
	ErrCannotHandle             = errors.New("cannot handle job")
)

type callback func(*sequence.ExecResult)

// dispatcher is responsible for "resolving" a sequence by sending messages to sats and collecting the results
type dispatcher struct {
	log       *vlog.Logger
	pod       *bus.Pod
	callbacks map[string]callback
	lock      *sync.RWMutex
}

type sequenceDispatcher struct {
	seq *sequence.Sequence
	pod *bus.Pod
	log *vlog.Logger
}

func newDispatcher(log *vlog.Logger, pod *bus.Pod) *dispatcher {
	d := &dispatcher{
		log:       log,
		pod:       pod,
		callbacks: make(map[string]callback),
		lock:      &sync.RWMutex{},
	}

	d.pod.OnType(MsgTypeSuborbitalResult, d.onMsgHandler())

	return d
}

// Execute returns the "final state" of a Sequence. If the state's err is not nil, it means a runnable returned an error, and the Directive indicates the Sequence should return.
// if exec itself actually returns an error other than ErrSequenceRunErr, it means there was a problem executing the Sequence as described, and should be treated as such.
func (d *dispatcher) Execute(seq *sequence.Sequence) error {
	s := &sequenceDispatcher{
		seq: seq,
		pod: d.pod,
		log: d.log,
	}

	resultChan := make(chan *sequence.ExecResult)
	cb := func(result *sequence.ExecResult) {
		resultChan <- result
	}

	d.lock.Lock()
	d.callbacks[seq.ParentID()] = cb
	d.lock.Unlock()

	defer func() {
		d.lock.Lock()
		delete(d.callbacks, seq.ParentID())
		d.lock.Unlock()
	}()

	firstStep := seq.NextStep()
	if firstStep == nil {
		return errors.New("sequence contains no steps")
	}

	if err := s.dispatchSingle(firstStep, resultChan); err != nil {
		return errors.Wrap(err, "failed to dispatchSingle")
	}

	for {
		// if there is only one step in the sequence, this loop will not run
		// but if additional steps exist, we need only await their responses
		// as the sats will handle dispatching each subsequence step themselves

		step := seq.NextStep()
		if step == nil {
			break
		} else if step.IsSingle() {
			if err := s.awaitResult(resultChan); err != nil {
				return errors.Wrap(err, "failed to awaitResult")
			}
		} else if step.IsGroup() {
			return errors.Wrap(ErrCannotHandle, "dispatching group steps not yet supported")
		}
	}

	return nil
}

// dispatchSingle executes a single plugin from a sequence step
func (s *sequenceDispatcher) dispatchSingle(step *sequence.Step, resultChan chan *sequence.ExecResult) error {
	data, err := s.seq.Request().ToJSON()
	if err != nil {
		return errors.Wrap(err, "failed to req.toJSON")
	}

	msg := bus.NewMsgWithParentID(step.FQMN, s.seq.ParentID(), data)

	// find an appropriate peer and tunnel the first excution to them
	if err := s.pod.Tunnel(step.FQMN, msg); err != nil {
		return errors.Wrap(err, "failed to Tunnel")
	}

	s.log.Debug("dispatched execution for", s.seq.ParentID(), "to peer with message", msg.UUID())

	return s.awaitResult(resultChan)
}

func (s *sequenceDispatcher) awaitResult(resultChan chan *sequence.ExecResult) error {
	select {
	case result := <-resultChan:
		if result.Response == nil {
			return fmt.Errorf("recieved nil response for %s", result.FQMN)
		}

		if err := s.seq.HandleStepResults([]sequence.ExecResult{*result}); err != nil {
			return errors.Wrap(err, "failed to HandleStepResults")
		}
	case <-time.After(time.Second * 10):
		return ErrDispatchTimeout
	}

	return nil
}

// onMsgHandler is called when a new message is received from the pod
func (d *dispatcher) onMsgHandler() bus.MsgFunc {
	return func(msg bus.Message) error {
		d.lock.RLock()
		defer d.lock.RUnlock()
		// we only care about the messages related to our specific sequence
		callback, exists := d.callbacks[msg.ParentID()]
		if !exists {
			return nil
		}

		result := &sequence.ExecResult{}

		if err := json.Unmarshal(msg.Data(), result); err != nil {
			d.log.Error(errors.Wrap(err, "failed to Unmarshal message data"))
			return nil
		}

		callback(result)

		return nil
	}
}
