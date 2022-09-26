package coordinator

import (
	"net/http"

	"github.com/suborbital/vektor/vk"
)

// nolint
func (c *Coordinator) authMiddleware() vk.Middleware {
	return func(r *http.Request, ctx *vk.Ctx) error {
		if c.opts.EnvironmentToken != "" {
			return nil
		} else {
			return vk.E(http.StatusUnauthorized, "unauthorized")
		}

		// TODO: restore the ability to have dynamic auth
		// FQFN, err := fqfn.FromURL(r.URL)
		// if err != nil {
		// 	ctx.Log.Debug(errors.Wrap(err, "failed to fqfn.FromURL, skipping headless auth"))
		// 	return nil
		// }

		// auth := r.Header.Get("Authorization")

		// // we call GetModule, which by now should have the module cached, so it'll be fast.
		// module, err := c.App.GetModule(FQFN)
		// if err != nil {
		// 	ctx.Log.Error(errors.Wrap(err, "failed to GetModule"))
		// 	return vk.E(http.StatusBadRequest, "invalid FQFN URI")
		// }

		// if len(module.TokenHash) > 0 {
		// 	providedHash := appsource.TokenHash(auth)

		// 	if subtle.ConstantTimeCompare(module.TokenHash, providedHash) != 1 {
		// 		ctx.Log.Error(errors.New("provided authorization header does not match module's token hash"))
		// 		return vk.E(http.StatusUnauthorized, "unauthorized")
		// 	}
		// }

		return nil
	}
}
