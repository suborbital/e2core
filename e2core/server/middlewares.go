package server

import (
	"net/http"

	"github.com/suborbital/vektor/vk"
)

type requestScope struct {
	RequestID string `json:"request_id"`
}

func scopeMiddleware(inner vk.HandlerFunc) vk.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, ctx *vk.Ctx) error {
		scope := requestScope{
			RequestID: ctx.RequestID(),
		}

		ctx.UseScope(scope)

		return inner(w, r, ctx)
	}
}
