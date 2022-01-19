package coordinator

import (
	"crypto/subtle"
	"net/http"

	"github.com/pkg/errors"
	"github.com/suborbital/atmo/atmo/appsource"
	"github.com/suborbital/atmo/fqfn"
	"github.com/suborbital/vektor/vk"
)

func (c *Coordinator) headlessAuthMiddleware() vk.Middleware {
	return func(r *http.Request, ctx *vk.Ctx) error {
		FQFN, err := fqfn.FromURL(r.URL)
		if err != nil {
			ctx.Log.Debug(errors.Wrap(err, "failed to fqfn.FromURL, skipping headless auth"))
			return nil
		}

		auth := r.Header.Get("Authorization")

		// we call FindRunnable, which by now should have the Runnable cached, so it'll be fast
		runnable, err := c.App.FindRunnable(FQFN, auth)
		if err != nil {
			ctx.Log.Error(errors.Wrap(err, "failed to FindRunnable"))
			return vk.E(http.StatusBadRequest, "invalid FQFN URI")
		}

		if len(runnable.TokenHash) > 0 {
			providedHash := appsource.TokenHash(auth)

			if subtle.ConstantTimeCompare(runnable.TokenHash, providedHash) != 1 {
				ctx.Log.Error(errors.New("provided authorization header does not match runnable's token hash"))
				return vk.E(http.StatusUnauthorized, "unauthorized")
			}
		}

		return nil
	}
}
