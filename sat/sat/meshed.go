package sat

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/suborbital/e2core/e2core/sequence"
	"github.com/suborbital/e2core/e2core/server"
	"github.com/suborbital/e2core/foundation/bus/bus"
	"github.com/suborbital/e2core/foundation/scheduler"
	"github.com/suborbital/systemspec/request"
)

// handleFnResult is mounted onto exec.ListenAndRun...
// when a meshed peer sends us a job, it is executed by Reactr and then
// the result is passed into this function for handling
func (s *Sat) handleFnResult(msg bus.Message, result interface{}, fnErr error) {
	ll := s.logger.With().Str("method", "handleFnResult").Logger()

	fmt.Println(string(msg.Data()))

	// first unmarshal the request and sequence information
	req, err := request.FromJSON(msg.Data())
	if err != nil {
		ll.Err(err).Msg("request.FromJSON")
		return
	}

	ctx := context.WithValue(context.Background(), "requestID", req.ID)

	spanCtx, span := s.tracer.Start(ctx, "handleFnResult", trace.WithAttributes(
		attribute.String("request_id", req.ID),
	))
	defer span.End()

	seq, err := sequence.FromJSON(req.SequenceJSON, req)
	if err != nil {
		ll.Err(err).Msg("sequence.FromJSON")
		return
	}

	// figure out where we are in the sequence
	step := seq.NextStep()
	if step == nil {
		ll.Error().Msg("got nil seq.NextStep")
		return
	}

	// start evaluating the result of the function call
	resp := &request.CoordinatedResponse{}
	var runErr scheduler.RunErr
	var execErr error

	if fnErr != nil {
		if fnRunErr, isRunErr := fnErr.(scheduler.RunErr); isRunErr {
			// great, it's a runErr
			runErr = fnRunErr
		} else {
			execErr = fnErr
		}
	} else {
		resp = result.(*request.CoordinatedResponse)
	}

	// package everything up and shuttle it back to the parent (e2core)
	fnr := &sequence.ExecResult{
		FQMN:     msg.Type(),
		Response: resp,
		RunErr:   runErr,
		ExecErr: func() string {
			if execErr != nil {
				return execErr.Error()
			}

			return ""
		}(),
	}

	if err = s.sendFnResult(fnr, spanCtx); err != nil {
		ll.Err(err).Msg("s.sendFnResult")
		return
	}

	// determine if we ourselves should continue or halt the sequence
	if execErr != nil {
		ll.Err(execErr).Str("messageType", msg.Type()).Msg("stopping execution after exec error")
		return
	}

	if err = seq.HandleStepResults([]sequence.ExecResult{*fnr}); err != nil {
		ll.Err(err).Msg("seq.HandleStepResults")
		return
	}

	// prepare for the next step in the chain to be executed
	stepJSON, err := seq.StepsJSON()
	if err != nil {
		ll.Err(err).Msg("seq.StepsJSON")
		return
	}

	req.SequenceJSON = stepJSON

	s.sendNextStep(msg, seq, req, spanCtx)
}

func (s *Sat) sendFnResult(result *sequence.ExecResult, ctx context.Context) error {
	span := trace.SpanFromContext(ctx)
	defer span.End()

	fnrJSON, err := json.Marshal(result)
	if err != nil {
		return errors.Wrap(err, "failed to Marshal function result")
	}

	reqID, ok := ctx.Value("requestID").(string)
	if !ok {
		return errors.New("request ID was not in the context")
	}

	respMsg := bus.NewMsgWithParentID(server.MsgTypeSuborbitalResult, reqID, fnrJSON)

	s.logger.Debug().
		Str("method", "sendFnResult").
		Str("function", s.config.JobType).
		Str("respUUID", respMsg.UUID()).
		Msg("function completed, sending meshed result message")

	if s.pod.Send(respMsg) == nil {
		return errors.New("failed to Send fnResult")
	}

	return nil
}

func (s *Sat) sendNextStep(_ bus.Message, seq *sequence.Sequence, req *request.CoordinatedRequest, ctx context.Context) {
	ll := s.logger.With().Str("method", "sendNextStep").Logger()

	span := trace.SpanFromContext(ctx)
	defer span.End()

	nextStep := seq.NextStep()
	if nextStep == nil {
		ll.Debug().Msg("sequence completed, no nextStep message to send")
		return
	}

	reqJSON, err := json.Marshal(req)
	if err != nil {
		ll.Err(err).Msg("json.Marshal request")
		return
	}

	reqID, ok := ctx.Value("requestID").(string)
	if !ok {
		ll.Error().Msg("request ID was not present in the context, stopping")
		return
	}

	nextMsg := bus.NewMsgWithParentID(nextStep.FQMN, reqID, reqJSON)

	ll.Debug().Str("nextStep", nextStep.FQMN).Str("nextMessage", nextMsg.UUID()).Msg("sending next message")

	if err := s.bus.Tunnel(nextStep.FQMN, nextMsg); err != nil {
		// nothing much we can do here
		ll.Err(err).Msg("bus.Tunnel next step")
	}
}
