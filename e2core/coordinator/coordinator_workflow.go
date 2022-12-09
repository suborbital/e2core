package coordinator

import (
	"net/http"

	"github.com/pkg/errors"

	"github.com/suborbital/e2core/e2core/coordinator/sequence"
	"github.com/suborbital/e2core/foundation/scheduler"
	"github.com/suborbital/systemspec/request"
	"github.com/suborbital/systemspec/tenant"
	"github.com/suborbital/vektor/vk"
)

func (c *Coordinator) vkHandlerForWorkflow(wfl tenant.Workflow) vk.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, ctx *vk.Ctx) error {
		req, err := request.FromVKRequest(r, ctx)
		if err != nil {
			ctx.Log.Error(errors.Wrap(err, "failed to request.FromVKRequest"))
			return vk.E(http.StatusInternalServerError, "failed to handle request")
		}

		req.UseSuborbitalHeaders(r, ctx)

		// a sequence executes the handler's steps and manages its state.
		seq, err := sequence.New(wfl.Steps, req, ctx)
		if err != nil {
			ctx.Log.Error(errors.Wrap(err, "failed to sequence.New"))
			return vk.E(http.StatusInternalServerError, "failed to handle request")
		}

		if err := seq.Execute(c.exec); err != nil {
			ctx.Log.Error(errors.Wrap(err, "failed to seq.exec"))

			if runErr, isRunErr := err.(scheduler.RunErr); isRunErr {
				if runErr.Code < 200 || runErr.Code > 599 {
					// if the module returned an invalid code for HTTP, default to 500.
					return vk.Err(http.StatusInternalServerError, runErr.Message)
				}

				return vk.Err(runErr.Code, runErr.Message)
			}

			return vk.Wrap(http.StatusInternalServerError, err)
		}

		// handle any response headers that were set by the Runnables.
		if req.RespHeaders != nil {
			for head, val := range req.RespHeaders {
				ctx.RespHeaders.Set(head, val)
			}
		}

		return vk.RespondBytes(ctx.Context, w, resultFromState(wfl.Steps, req.State), http.StatusOK)
	}
}
