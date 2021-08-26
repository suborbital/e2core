package coordinator

import (
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"
	"github.com/suborbital/atmo/directive"
	"github.com/suborbital/reactr/request"
	"github.com/suborbital/vektor/vk"
)

func (c *Coordinator) vkHandlerForDirectiveHandler(handler directive.Handler) vk.HandlerFunc {
	return func(r *http.Request, ctx *vk.Ctx) (interface{}, error) {
		req, err := request.FromVKRequest(r, ctx)
		if err != nil {
			ctx.Log.Error(errors.Wrap(err, "failed to request.FromVKRequest"))
			return nil, vk.E(http.StatusInternalServerError, "failed to handle request")
		}

		// this should probably be factored out into the CoordinateRequest object
		if *c.opts.Headless {
			// fill in initial state from the state header
			if stateJSON := r.Header.Get(atmoHeadlessStateHeader); stateJSON != "" {
				state := map[string]string{}
				byteState := map[string][]byte{}

				if err := json.Unmarshal([]byte(stateJSON), &state); err != nil {
					c.log.Error(errors.Wrap(err, "failed to Unmarshal X-Atmo-State header"))
				} else {
					// iterate over the state and convert each field to bytes
					for k, v := range state {
						byteState[k] = []byte(v)
					}
				}

				req.State = byteState
			}

			// fill in the URL params from the Params header
			if paramsJSON := r.Header.Get(atmoHeadlessParamsHeader); paramsJSON != "" {
				params := map[string]string{}

				if err := json.Unmarshal([]byte(paramsJSON), &params); err != nil {
					c.log.Error(errors.Wrap(err, "failed to Unmarshal X-Atmo-Params header"))
				} else {
					req.Params = params
				}
			}

			// add the request ID as a response header
			ctx.RespHeaders.Add(atmoRequestIDHeader, ctx.RequestID())
		}

		// a sequence executes the handler's steps and manages its state
		seq := newSequence(handler.Steps, c.exec, ctx)

		seqState, err := seq.execute(req)
		if err != nil {
			ctx.Log.Error(errors.Wrap(err, "failed to seq.exec"))

			if errors.Is(err, ErrSequenceRunErr) && seqState.err != nil {
				if seqState.err.Code < 200 || seqState.err.Code > 599 {
					// if the Runnable returned an invalid code for HTTP, default to 500
					return nil, vk.Err(http.StatusInternalServerError, seqState.err.Message)
				}

				return nil, vk.Err(seqState.err.Code, seqState.err.Message)
			}

			return nil, vk.Wrap(http.StatusInternalServerError, err)
		}

		// handle any response headers that were set by the Runnables
		if req.RespHeaders != nil {
			for head, val := range req.RespHeaders {
				ctx.RespHeaders.Set(head, val)
			}
		}

		return resultFromState(handler, seqState.state), nil
	}
}
