package sat

import (
	"net/http"

	"github.com/suborbital/e2core/foundation/scheduler"
	"github.com/suborbital/vektor/vk"
)

type WorkerMetricsResponse struct {
	Scheduler scheduler.ScalerMetrics `json:"scheduler"`
}

func (s *Sat) workerMetricsHandler() vk.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, ctx *vk.Ctx) error {
		metrics := s.engine.Metrics()

		resp := &WorkerMetricsResponse{
			Scheduler: metrics,
		}

		return vk.RespondJSON(ctx.Context, w, resp, http.StatusOK)
	}
}
