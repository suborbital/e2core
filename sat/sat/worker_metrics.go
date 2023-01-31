package sat

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/suborbital/e2core/foundation/scheduler"
)

type WorkerMetricsResponse struct {
	Scheduler scheduler.ScalerMetrics `json:"scheduler"`
}

func (s *Sat) workerMetricsHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		metrics := s.engine.Metrics()

		resp := &WorkerMetricsResponse{
			Scheduler: metrics,
		}

		return c.JSON(http.StatusOK, resp)
	}
}
