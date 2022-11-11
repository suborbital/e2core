package sat

import (
	"net/http"

	"github.com/pkg/errors"

	"github.com/suborbital/e2core/scheduler"
	"github.com/suborbital/vektor/vk"
)

type WorkerMetricsResponse struct {
	Scheduler scheduler.ScalerMetrics `json:"scheduler"`
}

func (s *Sat) workerMetricsHandler() vk.HandlerFunc {
	return func(r *http.Request, ctx *vk.Ctx) (interface{}, error) {
		metrics, err := s.exec.Metrics()
		if err != nil {
			ctx.Log.Error(errors.Wrap(err, "failed to exec.Metrics"))
			return nil, vk.E(http.StatusInternalServerError, "unknown error")
		}

		resp := &WorkerMetricsResponse{
			Scheduler: *metrics,
		}

		return resp, nil
	}
}
