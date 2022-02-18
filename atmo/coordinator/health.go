package coordinator

import (
	"net/http"

	"github.com/suborbital/vektor/vk"
)

func (c *Coordinator) health() vk.HandlerFunc {
	return func(request *http.Request, ctx *vk.Ctx) (interface{}, error) {
		return map[string]bool{"healthy": true}, nil
	}
}
