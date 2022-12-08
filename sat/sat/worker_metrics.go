package sat

import (
	"net/http"

	"github.com/pkg/errors"

	"github.com/suborbital/e2core/foundation/scheduler"
	"github.com/suborbital/vektor/vk"
)

type WorkerMetricsResponse struct {
	Scheduler scheduler.ScalerMetrics `json:"scheduler"`
}

func (s *Sat) workerMetricsHandler() vk.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, ctx *vk.Ctx) error {
		metrics, err := s.exec.Metrics()
		if err != nil {
			ctx.Log.Error(errors.Wrap(err, "failed to exec.Metrics"))
			return vk.E(http.StatusInternalServerError, "unknown error")
		}

		resp := &WorkerMetricsResponse{
			Scheduler: *metrics,
		}

		return vk.RespondJSON(ctx.Context, w, resp, http.StatusOK)
	}
}
