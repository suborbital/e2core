package coordinator

import (
	"net/http"

	"github.com/pkg/errors"

	"github.com/suborbital/appspec/request"
	"github.com/suborbital/appspec/tenant/executable"
	"github.com/suborbital/e2core/scheduler"
	"github.com/suborbital/e2core/server/coordinator/sequence"
	"github.com/suborbital/vektor/vk"
)

func (c *Coordinator) vkHandlerForModuleByName() vk.HandlerFunc {
	return func(r *http.Request, ctx *vk.Ctx) (interface{}, error) {
		ident := ctx.Params.ByName("ident")
		namespace := ctx.Params.ByName("namespace")
		name := ctx.Params.ByName("name")

		mod := c.syncer.GetModuleByName(ident, namespace, name)
		if mod == nil {
			return nil, vk.E(http.StatusNotFound, "module not found")
		}

		req, err := request.FromVKRequest(r, ctx)
		if err != nil {
			ctx.Log.Error(errors.Wrap(err, "failed to request.FromVKRequest"))
			return nil, vk.E(http.StatusInternalServerError, "failed to handle request")
		}

		req.UseSuborbitalHeaders(r, ctx)

		steps := []executable.Executable{
			{
				ExecutableMod: executable.ExecutableMod{
					FQMN: mod.FQMN,
				},
			},
		}

		// a sequence executes the handler's steps and manages its state.
		seq, err := sequence.New(steps, req, ctx)
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

		return resultFromState(steps, req.State), nil
	}
}

func (c *Coordinator) vkHandlerForModuleByRef() vk.HandlerFunc {
	return func(r *http.Request, ctx *vk.Ctx) (interface{}, error) {
		ref := ctx.Params.ByName("ref")

		mod := c.syncer.GetModuleByRef(ref)
		if mod == nil {
			return nil, vk.E(http.StatusNotFound, "module not found")
		}

		ctx.Log.Debug("found module by ref:", mod.FQMN, mod.Name, mod.Namespace)

		req, err := request.FromVKRequest(r, ctx)
		if err != nil {
			ctx.Log.Error(errors.Wrap(err, "failed to request.FromVKRequest"))
			return nil, vk.E(http.StatusInternalServerError, "failed to handle request")
		}

		if err := req.UseSuborbitalHeaders(r, ctx); err != nil {
			return nil, vk.E(http.StatusBadRequest, "bad request")
		}

		steps := []executable.Executable{
			{
				ExecutableMod: executable.ExecutableMod{
					FQMN: mod.FQMN,
				},
			},
		}

		// a sequence executes the handler's steps and manages its state.
		seq, err := sequence.New(steps, req, ctx)
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

		return resultFromState(steps, req.State), nil
	}
}
