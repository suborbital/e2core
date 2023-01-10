package sat

import (
	"net/http"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/suborbital/e2core/foundation/scheduler"
	"github.com/suborbital/e2core/sat/engine2"
	"github.com/suborbital/e2core/sat/sat/metrics"
	"github.com/suborbital/systemspec/request"
	"github.com/suborbital/vektor/vk"
)

func (s *Sat) handler(engine *engine2.Engine) vk.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, ctx *vk.Ctx) error {
		spanCtx, span := s.tracer.Start(ctx.Context, "vkhandler", trace.WithAttributes(
			attribute.String("request_id", ctx.RequestID()),
		))
		defer span.End()

		s.metrics.FunctionExecutions.Add(spanCtx, 1)

		ctx.Context = spanCtx

		req, err := request.FromVKRequest(r, ctx)
		if err != nil {
			ctx.Log.Error(errors.Wrap(err, "failed to FromVKRequest"))
			return vk.E(http.StatusInternalServerError, "unknown error")
		}

		t := metrics.NewTimer()

		var runErr scheduler.RunErr

		result, err := engine.Do(scheduler.NewJob(s.jobName, req)).Then()
		if err != nil {
			if errors.As(err, &runErr) {
				// runErr would be an actual error returned from a function
				// should find a better way to determine if a RunErr is "non-nil"
				if runErr.Code != 0 || runErr.Message != "" {
					s.config.Logger.Debug("fn", s.jobName, "returned an error")
					return vk.E(runErr.Code, runErr.Message)
				}
			}

			s.config.Logger.Error(errors.Wrap(err, "failed to exec.Do"))
			return vk.E(http.StatusInternalServerError, "unknown error")
		}

		s.metrics.FunctionTime.Record(spanCtx, t.Observe(), attribute.String("id", req.ID))

		if result == nil {
			s.config.Logger.Debug("fn", s.jobName, "returned a nil result")
			return nil
		}

		resp := result.(*request.CoordinatedResponse)

		for headerKey, headerValue := range resp.RespHeaders {
			ctx.RespHeaders.Set(headerKey, headerValue)
		}

		return vk.RespondBytes(ctx.Context, w, resp.Output, http.StatusOK)
	}
}
