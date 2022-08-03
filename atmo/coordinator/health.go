package coordinator

import (
	"net/http"

	"github.com/suborbital/vektor/vk"
)

func (c *Coordinator) health() vk.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request, ctx *vk.Ctx) error {
		return vk.RespondJSON(ctx.Context, w, map[string]bool{"healthy": true}, http.StatusOK)
	}
}
