package coordinator

import (
	"net/http"

	"github.com/pkg/errors"

	"github.com/suborbital/deltav/directive"
	"github.com/suborbital/deltav/scheduler"
	"github.com/suborbital/deltav/server/coordinator/sequence"
	"github.com/suborbital/deltav/server/request"
	"github.com/suborbital/vektor/vk"
)

func (c *Coordinator) vkHandlerForDirectiveHandler(handler directive.Handler) vk.HandlerFunc {
	return func(r *http.Request, ctx *vk.Ctx) (interface{}, error) {
		req, err := request.FromVKRequest(r, ctx)
		if err != nil {
			ctx.Log.Error(errors.Wrap(err, "failed to request.FromVKRequest"))
			return nil, vk.E(http.StatusInternalServerError, "failed to handle request")
		}

		// Pull the X-Atmo-State and X-Atmo-Params headers into the request.
		if *c.opts.Headless {
			req.UseHeadlessHeaders(r, ctx)
		}

		// a sequence executes the handler's steps and manages its state.
		seq, err := sequence.New(handler.Steps, req, ctx)
		if err != nil {
			ctx.Log.Error(errors.Wrap(err, "failed to sequence.New"))
			return nil, vk.E(http.StatusInternalServerError, "failed to handle request")
		}

		if err := seq.Execute(c.exec); err != nil {
			ctx.Log.Error(errors.Wrap(err, "failed to seq.exec"))

			if runErr, isRunErr := err.(scheduler.RunErr); isRunErr {
				if runErr.Code < 200 || runErr.Code > 599 {
					// if the Runnable returned an invalid code for HTTP, default to 500.
					return nil, vk.Err(http.StatusInternalServerError, runErr.Message)
				}

				return nil, vk.Err(runErr.Code, runErr.Message)
			}

			return nil, vk.Wrap(http.StatusInternalServerError, err)
		}

		// handle any response headers that were set by the Runnables.
		if req.RespHeaders != nil {
			for head, val := range req.RespHeaders {
				ctx.RespHeaders.Set(head, val)
			}
		}

		return resultFromState(handler, req.State), nil
	}
}
