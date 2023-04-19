package server

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/suborbital/e2core/e2core/sequence"
	"github.com/suborbital/systemspec/request"
	"github.com/suborbital/systemspec/tenant"
)

func (s *Server) executePluginByNameHandler() echo.HandlerFunc {
	return func(c echo.Context) error {

		// with the authorization middleware, this is going to be the uuid of the tenant specified by the path name in
		// the environment specified by the authorization token.
		ident := ReadParam(c, "ident")

		// this is coming from the path.
		namespace := ReadParam(c, "namespace")

		// this is coming from the path.
		name := ReadParam(c, "name")

		ll := s.logger.With().
			Str("ident", ident).
			Str("namespace", namespace).
			Str("fn", name).
			Logger()

		mod := s.syncer.GetModuleByName(ident, namespace, name)
		if mod == nil {
			ll.Error().Msg("syncer did not find module by these details")
			return echo.NewHTTPError(http.StatusNotFound, "module not found").SetInternal(fmt.Errorf("no module with %s/%s/%s", ident, namespace, name))
		}

		req, err := request.FromEchoContext(c)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to handle request").SetInternal(err)
		}

		err = req.UseSuborbitalHeaders(c)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to handle request").SetInternal(err)
		}

		ll.Info().
			Str("fqmn", mod.FQMN).
			Msg("found module with fqmn")

		steps := []tenant.WorkflowStep{{FQMN: mod.FQMN}}

		// a sequence executes the handler's steps and manages its state.
		seq, err := sequence.New(steps, req)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to handle request").SetInternal(err)
		}

		if err := s.dispatcher.Execute(seq); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to execute plugin").SetInternal(err)
		}

		// handle any response headers that were set by the Runnables.
		if req.RespHeaders != nil {
			for head, val := range req.RespHeaders {
				// need to directly assign because .Add and .Set will filter out non-standard
				// header names, which ours are.
				if c.Response().Header()[head] == nil {
					c.Response().Header()[head] = make([]string, 0)
				}

				c.Response().Header()[head] = append(c.Response().Header()[head], val)
			}
		}

		responseData := seq.Request().State[mod.FQMN]

		ll.Info().Str("fqmn", mod.FQMN).Msg("finished execution of the module, sending back data")

		return c.Blob(http.StatusOK, "application/octet-stream", responseData)
	}
}

func (s *Server) healthHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]bool{"healthy": true})
	}
}

// ReadParam tries to grab the value by name from the echo context first, and if it doesn't find it, then it falls back
// onto the path parameter.
func ReadParam(ctx echo.Context, name string) string {
	v := ctx.Get(name)
	if v != nil {
		return v.(string)
	}

	return ctx.Param(name)
}
