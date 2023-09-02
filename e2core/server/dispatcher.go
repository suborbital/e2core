package server

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/suborbital/e2core/e2core/sequence"
	"github.com/suborbital/e2core/foundation/bus/bus"
	"github.com/suborbital/e2core/foundation/tracing"
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
	log       zerolog.Logger
	pod       *bus.Pod
	callbacks map[string]callback
	lock      *sync.RWMutex
}

type sequenceDispatcher struct {
	seq *sequence.Sequence
	pod *bus.Pod
	log zerolog.Logger
}

func newDispatcher(l zerolog.Logger, pod *bus.Pod) *dispatcher {
	ll := l.With().Str("module", "dispatcher").Logger()
	d := &dispatcher{
		log:       ll,
		pod:       pod,
		callbacks: make(map[string]callback),
		lock:      &sync.RWMutex{},
	}

	d.pod.OnType(MsgTypeSuborbitalResult, d.onMsgHandler())

	return d
}

// Execute returns the "final state" of a Sequence. If the state's err is not nil, it means a runnable returned an error, and the Directive indicates the Sequence should return.
// if exec itself actually returns an error other than ErrSequenceRunErr, it means there was a problem executing the Sequence as described, and should be treated as such.
func (d *dispatcher) Execute(ctx context.Context, seq *sequence.Sequence) error {
	ctx, span := tracing.Tracer.Start(ctx, "dispatcher.execute")
	defer span.End()

	ll := d.log.With().Str("requestID", seq.ParentID()).Logger()
	ll.Info().Interface("dispatcher-pod", d.pod).Msg("created a sequence dispatcher")
	s := &sequenceDispatcher{
		seq: seq,
		pod: d.pod,
		log: ll.With().Str("part", "sequenceDispatcher").Logger(),
	}

	ll.Info().Msg("creating a result chan, and a callback function that takes in a result, and sends that result back into the resultchan.")

	resultChan := make(chan *sequence.ExecResult)
	cb := func(result *sequence.ExecResult) {
		ll.Info().Msg("callback: sending result to resultchan")
		resultChan <- result
	}

	ll.Info().Msg("this callback is added to the sequence.parentID in the dispatcher. It's just a map. One sequence ID, one callback")
	d.addCallback(seq.ParentID(), cb)
	defer d.removeCallback(seq.ParentID())

	firstStep := seq.NextStep()
	if firstStep == nil {
		return errors.New("sequence contains no steps")
	}

	ll.Info().Interface("first-step", firstStep).Msg("dispatchsingle gets called on the sequence dispatcher. Arguments are the results channel and the first step.")

	span.AddEvent("dispatching single")
	if err := s.dispatchSingle(ctx, firstStep, resultChan); err != nil {
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
			if err := s.awaitResult(ctx, resultChan); err != nil {
				return errors.Wrap(err, "failed to awaitResult")
			}
		} else if step.IsGroup() {
			return errors.Wrap(ErrCannotHandle, "dispatching group steps not yet supported")
		}
	}

	return nil
}

// dispatchSingle executes a single plugin from a sequence step
func (s *sequenceDispatcher) dispatchSingle(ctx context.Context, step *sequence.Step, resultChan chan *sequence.ExecResult) error {
	ctx, span := tracing.Tracer.Start(ctx, "sequencedispatcher.dispatchsingle")
	defer span.End()

	data, err := s.seq.Request().ToJSON()
	if err != nil {
		return errors.Wrap(err, "failed to req.toJSON")
	}

	s.log.Info().Str("data in dispatchSingle", string(data)).Msg("message about to be sent")

	span.AddEvent("created new message with parent id")
	msg := bus.NewMsgWithParentID(step.FQMN, s.seq.ParentID(), data)
	msg.SetContext(ctx)

	s.log.Info().Interface("bus.Message", msg).Msg("bus msg. Next is pod.tunnel with step.fqmn with message.")

	// find an appropriate peer and tunnel the first execution to them
	if err := s.pod.Tunnel(step.FQMN, msg); err != nil {
		return errors.Wrap(err, "failed to Tunnel")
	}

	s.log.Info().
		Str("msgUUID", msg.UUID()).
		Msg("dispatched execution for parent to peer with message")

	return s.awaitResult(ctx, resultChan)
}

func (s *sequenceDispatcher) awaitResult(ctx context.Context, resultChan chan *sequence.ExecResult) error {
	ctx, span := tracing.Tracer.Start(ctx, "awaitResult")
	defer span.End()

	select {
	case result := <-resultChan:
		span.AddEvent("result came in the channel")

		s.log.Info().Msg("we have a message back from the result channel")
		if result.Response == nil {
			s.log.Error().Msg("sadly the response was nil")
			return fmt.Errorf("recieved nil response for %s", result.FQMN)
		}

		s.log.Info().Msg("handling the step results")
		if err := s.seq.HandleStepResults([]sequence.ExecResult{*result}); err != nil {
			s.log.Err(err).Msg("something went wrong while handling the step results")
			return errors.Wrap(err, "failed to HandleStepResults")
		}
	case <-time.After(time.Second * 10):
		span.AddEvent("10 seconds have passed, sad times")

		s.log.Warn().Msg("dispatchSingle timeout reached")
		return ErrDispatchTimeout
	}

	return nil
}

// onMsgHandler is called when a new message is received from the pod
func (d *dispatcher) onMsgHandler() bus.MsgFunc {
	return func(msg bus.Message) error {
		ll := d.log.With().Str("requestID", msg.ParentID()).Logger()
		d.lock.RLock()
		defer d.lock.RUnlock()

		ll.Info().Msg("message received to dispatcher.onMsgHandler")

		// we only care about the messages related to our specific sequence
		callback, exists := d.callbacks[msg.ParentID()]
		if !exists {
			ll.Warn().Str("uuid", msg.ParentID()).Msg("did not exist")
			return nil
		}

		result := &sequence.ExecResult{}

		if err := json.Unmarshal(msg.Data(), result); err != nil {
			ll.Err(err).Msg("json.Unmarshal message data failure")
			return nil
		}

		ll.Info().Str("requestID", msg.ParentID()).Msg("calling the callback with the result")

		callback(result)

		return nil
	}
}

func (d *dispatcher) addCallback(parentID string, cb callback) {
	d.lock.Lock()
	defer d.lock.Unlock()

	d.callbacks[parentID] = cb
}

func (d *dispatcher) removeCallback(parentID string) {
	d.lock.Lock()
	defer d.lock.Unlock()

	delete(d.callbacks, parentID)
}
