package server

import (
	"encoding/json"
	"fmt"
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

type callback func(sequence.ExecResult)

// dispatcher is responsible for "resolving" a sequence by sending messages to sats and collecting the results
type dispatcher struct {
	log *vlog.Logger
	pod *bus.Pod
	seq *sequence.Sequence
}

func newDispatcher(log *vlog.Logger, pod *bus.Pod, sequence *sequence.Sequence) *dispatcher {
	d := &dispatcher{
		log: log,
		pod: pod,
		seq: sequence,
	}

	return d
}

// Execute returns the "final state" of a Sequence. If the state's err is not nil, it means a runnable returned an error, and the Directive indicates the Sequence should return.
// if exec itself actually returns an error other than ErrSequenceRunErr, it means there was a problem executing the Sequence as described, and should be treated as such.
func (d *dispatcher) Execute() error {
	resultChan := make(chan *sequence.ExecResult)
	d.pod.OnType(MsgTypeSuborbitalResult, d.onMsgHandler(resultChan))

	for {
		// continue running steps until the sequence is complete.
		step := d.seq.NextStep()
		if step == nil {
			break
		}

		if step.IsSingle() {
			if err := d.executeSingle(step, resultChan); err != nil {
				return errors.Wrap(err, "failed to executeSingle")
			}
		} else if step.IsGroup() {
			return errors.Wrap(ErrCannotHandle, "dispatching group steps not yet supported")
		}
	}

	return nil
}

// executeSingle executes a single plugin from a sequence step
func (d *dispatcher) executeSingle(step *sequence.Step, resultChan chan *sequence.ExecResult) error {
	data, err := d.seq.Request().ToJSON()
	if err != nil {
		return errors.Wrap(err, "failed to req.toJSON")
	}

	msg := bus.NewMsgWithParentID(step.FQMN, d.seq.ParentID(), data)

	// find an appropriate peer and tunnel the first excution to them
	if err := d.pod.Tunnel(step.FQMN, msg); err != nil {
		return errors.Wrap(err, "failed to Tunnel")
	}

	d.log.Debug("dispatched execution for", d.seq.ParentID(), "to peer with message", msg.UUID())

	// wait until the sequence completes or errors
	select {
	case result := <-resultChan:
		if result.Response == nil {
			return fmt.Errorf("recieved nil response for %s", result.FQMN)
		}

		if err := d.seq.HandleStepResults([]sequence.ExecResult{*result}); err != nil {
			return errors.Wrap(err, "failed to HandleStepResults")
		}
	case <-time.After(time.Second * 10):
		return ErrDispatchTimeout
	}

	return nil
}

// onMsgHandler is called when a new message is received from the pod
func (d *dispatcher) onMsgHandler(resultChan chan *sequence.ExecResult) bus.MsgFunc {
	return func(msg bus.Message) error {
		// we only care about the messages related to our specific sequence
		if msg.ParentID() != d.seq.ParentID() {
			return nil
		}

		result := &sequence.ExecResult{}

		if err := json.Unmarshal(msg.Data(), result); err != nil {
			// nothing really to be done here
			return nil
		}

		resultChan <- result

		return nil
	}
}
