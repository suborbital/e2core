package server

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"

	"github.com/suborbital/e2core/e2core/sequence"
	"github.com/suborbital/systemspec/request"
	"github.com/suborbital/systemspec/tenant"
)

func (s *Server) executePluginByNameHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		ident := c.Param("ident")
		namespace := c.Param("namespace")
		name := c.Param("name")

		mod := s.syncer.GetModuleByName(ident, namespace, name)
		if mod == nil {
			return echo.NewHTTPError(http.StatusNotFound, "module not found")
		}

		req, err := request.FromEchoContext(c)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to handle request").SetInternal(err)
		}

		err = req.UseSuborbitalHeaders(c)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to handle request").SetInternal(err)
		}

		steps := []tenant.WorkflowStep{{FQMN: mod.FQMN}}

		// a sequence executes the handler's steps and manages its state.
		seq, err := sequence.New(steps, req)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to handle request")
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

		return c.Blob(http.StatusOK, "text/plain", responseData)
	}
}

func (s *Server) executePluginByRefHandler(l zerolog.Logger) echo.HandlerFunc {
	ll := l.With().Str("handler", "executePluginByRefHandler").Logger()

	return func(c echo.Context) error {
		ref := c.Param("ref")

		mod := s.syncer.GetModuleByRef(ref)
		if mod == nil {
			return echo.NewHTTPError(http.StatusNotFound, "module not found")
		}

		ll.Debug().Str("fqmn", mod.FQMN).Msg("found module by ref")

		req, err := request.FromEchoContext(c)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError).SetInternal(err)
		}

		err = req.UseSuborbitalHeaders(c)
		if err != nil {
			return err
		}

		steps := []tenant.WorkflowStep{{FQMN: mod.FQMN}}

		// a sequence executes the handler's steps and manages its state.
		seq, err := sequence.New(steps, req)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to handle request")
		}

		if err := s.dispatcher.Execute(seq); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to execute plugin")
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

		return c.Blob(http.StatusOK, "text/plain", responseData)
	}
}

func (s *Server) executeWorkflowHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		ident := c.Param("ident")
		namespace := c.Param("namespace")
		name := c.Param("name")

		tnt := s.syncer.TenantOverview(ident)
		if tnt == nil {
			return echo.NewHTTPError(http.StatusNotFound, "not found")
		}

		namespaces := []tenant.NamespaceConfig{tnt.Config.DefaultNamespace}
		namespaces = append(namespaces, tnt.Config.Namespaces...)

		var workflow *tenant.Workflow

		// yes, this is a dumb and slow way to do this but we'll optimize later

	OUTER:
		for i := range namespaces {
			ns := namespaces[i]
			if ns.Name != namespace {
				continue
			}

			for j := range ns.Workflows {
				wfl := ns.Workflows[j]

				if wfl.Name != name {
					continue
				}

				workflow = &wfl
				break OUTER
			}
		}

		if workflow == nil {
			return echo.NewHTTPError(http.StatusNotFound, "not found")
		}

		req, err := request.FromEchoContext(c)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to handle request").SetInternal(err)
		}

		err = req.UseSuborbitalHeaders(c)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError).SetInternal(err)
		}

		// a sequence executes the handler's steps and manages its state.
		seq, err := sequence.New(workflow.Steps, req)
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

		// this should be smarter eventually (i.e. handle last-step groups properly)
		responseData := seq.Request().State[workflow.Steps[len(workflow.Steps)-1].FQMN]

		return c.Blob(http.StatusOK, "text/plain", responseData)
	}
}

func (s *Server) healthHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]bool{"healthy": true})
	}
}
