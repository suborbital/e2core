package coordinator

import (
    "net/http"

    "github.com/pkg/errors"

    "github.com/suborbital/atmo/atmo/coordinator/sequence"
    "github.com/suborbital/atmo/directive"
    "github.com/suborbital/reactr/request"
    "github.com/suborbital/reactr/rt"
    "github.com/suborbital/vektor/vk"
)

func (c *Coordinator) vkHandlerForDirectiveHandler(handler directive.Handler) vk.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request, ctx *vk.Ctx) error {
        req, err := request.FromVKRequest(r, ctx)
        if err != nil {
            ctx.Log.Error(errors.Wrap(err, "failed to request.FromVKRequest"))
            return vk.E(http.StatusInternalServerError, "failed to handle request")
        }

        // Pull the X-Atmo-State and X-Atmo-Params headers into the request.
        if *c.opts.Headless {
            req.UseHeadlessHeaders(r, ctx)
        }

        // a sequence executes the handler's steps and manages its state.
        seq, err := sequence.New(handler.Steps, req, c.exec, ctx)
        if err != nil {
            ctx.Log.Error(errors.Wrap(err, "failed to sequence.New"))
            return vk.E(http.StatusInternalServerError, "failed to handle request")
        }

        if err := seq.Execute(); err != nil {
            ctx.Log.Error(errors.Wrap(err, "failed to seq.exec"))

            if runErr, isRunErr := err.(rt.RunErr); isRunErr {
                if runErr.Code < 200 || runErr.Code > 599 {
                    // if the Runnable returned an invalid code for HTTP, default to 500.
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

        return vk.RespondBytes(ctx.Context, w, resultFromState(handler, req.State), http.StatusOK)
    }
}
