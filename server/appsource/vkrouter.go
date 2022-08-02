package appsource

import (
	"fmt"
	"net/http"
	"os"

	"github.com/pkg/errors"

	"github.com/suborbital/deltav/fqfn"
	"github.com/suborbital/deltav/server/options"
	"github.com/suborbital/vektor/vk"
)

// AppSourceVKRouter is a helper struct to generate a VK router that can serve
// an HTTP AppSource based on an actual AppSource object.
type AppSourceVKRouter struct {
	appSource AppSource
	options   options.Options
}

// NewAppSourceVKRouter creates a new AppSourceVKRouter.
func NewAppSourceVKRouter(appSource AppSource, opts options.Options) *AppSourceVKRouter {
	h := &AppSourceVKRouter{
		appSource: appSource,
		options:   opts,
	}

	return h
}

// GenerateRouter generates a VK router that uses an AppSource to serve data.
func (a *AppSourceVKRouter) GenerateRouter() (*vk.Router, error) {
	if err := a.appSource.Start(a.options); err != nil {
		return nil, errors.Wrap(err, "failed to appSource.Start")
	}

	router := vk.NewRouter(a.options.Logger, "")

	router.GET("/runnables/:ident/:version", a.RunnablesHandler())
	router.GET("/runnable/:ident/:namespace/:fn/:version", a.FindRunnableHandler())
	router.GET("/handlers/:ident/:version", a.HandlersHandler())
	router.GET("/schedules/:ident/:version", a.SchedulesHandler())
	router.GET("/connections/:ident/:version", a.ConnectionsHandler())
	router.GET("/authentication/:ident/:version", a.AuthenticationHandler())
	router.GET("/capabilities/:ident/:namespace/:version", a.CapabilitiesHandler())

	// this is undefined right now. I'm not sure how to fetch one file without explicit ident / version info.
	router.GET("/file/:ident/:version/:filename", a.FileHandler())
	router.GET("/meta", a.MetaHandler())

	return router, nil
}

// RunnablesHandler is a handler to fetch Runnables.
func (a *AppSourceVKRouter) RunnablesHandler() vk.HandlerFunc {
	return func(r *http.Request, ctx *vk.Ctx) (interface{}, error) {
		ident := ctx.Params.ByName("ident")
		version := ctx.Params.ByName("version")

		return a.appSource.Runnables(ident, version), nil
	}
}

// FindRunnableHandler is a handler to find a single Runnable.
func (a *AppSourceVKRouter) FindRunnableHandler() vk.HandlerFunc {
	return func(r *http.Request, ctx *vk.Ctx) (interface{}, error) {
		ident := ctx.Params.ByName("ident")
		namespace := ctx.Params.ByName("namespace")
		fn := ctx.Params.ByName("fn")
		version := ctx.Params.ByName("version")

		fqfn := fqfn.FromParts(ident, namespace, fn, version)

		// auth header can be used to authenticate requests.
		auth := r.Header.Get("Authorization")

		runnable, err := a.appSource.FindRunnable(fqfn, auth)
		if err != nil {
			ctx.Log.Error(errors.Wrap(err, "failed to FindRunnable"))

			if errors.Is(err, ErrRunnableNotFound) {
				return nil, vk.Wrap(http.StatusNotFound, fmt.Errorf("failed to find Runnable %s", fqfn))
			} else if errors.Is(err, ErrAuthenticationFailed) {
				return nil, vk.E(http.StatusUnauthorized, "unauthorized")
			}

			return nil, vk.E(http.StatusInternalServerError, "something went wrong")
		}

		return runnable, nil
	}
}

// HandlersHandler is a handler to fetch Handlers.
func (a *AppSourceVKRouter) HandlersHandler() vk.HandlerFunc {
	return func(r *http.Request, ctx *vk.Ctx) (interface{}, error) {
		ident := ctx.Params.ByName("ident")
		version := ctx.Params.ByName("version")

		return a.appSource.Handlers(ident, version), nil
	}
}

// SchedulesHandler is a handler to fetch Schedules.
func (a *AppSourceVKRouter) SchedulesHandler() vk.HandlerFunc {
	return func(r *http.Request, ctx *vk.Ctx) (interface{}, error) {
		ident := ctx.Params.ByName("ident")
		version := ctx.Params.ByName("version")

		return a.appSource.Schedules(ident, version), nil
	}
}

// ConnectionsHandler is a handler to fetch Connection data.
func (a *AppSourceVKRouter) ConnectionsHandler() vk.HandlerFunc {
	return func(r *http.Request, ctx *vk.Ctx) (interface{}, error) {
		ident := ctx.Params.ByName("ident")
		version := ctx.Params.ByName("version")

		return a.appSource.Connections(ident, version), nil
	}
}

// AuthenticationHandler is a handler to fetch Authentication data.
func (a *AppSourceVKRouter) AuthenticationHandler() vk.HandlerFunc {
	return func(r *http.Request, ctx *vk.Ctx) (interface{}, error) {
		ident := ctx.Params.ByName("ident")
		version := ctx.Params.ByName("version")

		return a.appSource.Authentication(ident, version), nil
	}
}

// CapabilitiesHandler is a handler to fetch Capabilities data.
func (a *AppSourceVKRouter) CapabilitiesHandler() vk.HandlerFunc {
	return func(r *http.Request, ctx *vk.Ctx) (interface{}, error) {
		ident := ctx.Params.ByName("ident")
		namespace := ctx.Params.ByName("namespace")
		version := ctx.Params.ByName("version")

		return a.appSource.Capabilities(ident, namespace, version), nil
	}
}

// FileHandler is a handler to fetch Files.
func (a *AppSourceVKRouter) FileHandler() vk.HandlerFunc {
	return func(r *http.Request, ctx *vk.Ctx) (interface{}, error) {
		filename := ctx.Params.ByName("filename")
		ident := ctx.Params.ByName("ident")
		version := ctx.Params.ByName("version")

		fileBytes, err := a.appSource.File(ident, version, filename)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil, vk.E(http.StatusNotFound, "not found")
			}

			return nil, vk.E(http.StatusInternalServerError, "something went wrong")
		}

		return fileBytes, nil
	}
}

// MetaHandler is a handler to fetch Metadata.
func (a *AppSourceVKRouter) MetaHandler() vk.HandlerFunc {
	return func(r *http.Request, ctx *vk.Ctx) (interface{}, error) {
		return a.appSource.Applications(), nil
	}
}
