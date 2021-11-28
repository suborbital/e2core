package coordinator

import (
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

		// Pull the X-Atmo-State and X-Atmo-Params headers into the request
		if *c.opts.Headless {
			req.UseHeadlessHeaders(r, ctx)
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
