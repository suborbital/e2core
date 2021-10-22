package coordinator

import (
	"github.com/suborbital/vektor/vk"
	"net/http"
)

func (c *Coordinator) health() vk.HandlerFunc {
	return func(request *http.Request, ctx *vk.Ctx) (interface{}, error) {
		return map[string]string{"result": "success"}, nil
	}
}
