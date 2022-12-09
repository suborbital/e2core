package coordinator

import (
	"net/http"

	"github.com/pkg/errors"

	"github.com/suborbital/e2core/foundation/scheduler"
	"github.com/suborbital/vektor/vk"
)

type MetricsResponse struct {
	Scheduler scheduler.ScalerMetrics `json:"scheduler"`
}

func (c *Coordinator) metricsHandler() vk.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, ctx *vk.Ctx) error {
		metrics, err := c.exec.Metrics()
		if err != nil {
			ctx.Log.Error(errors.Wrap(err, "failed to exec.Metrics"))
			return vk.E(http.StatusInternalServerError, "unknown error")
		}

		resp := &MetricsResponse{
			Scheduler: *metrics,
		}

		return vk.RespondJSON(ctx.Context, w, resp, http.StatusOK)
	}
}
