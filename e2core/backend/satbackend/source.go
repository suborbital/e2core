package satbackend

import (
	"net/http"
	"os"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/suborbital/systemspec/fqmn"
	"github.com/suborbital/systemspec/system"
)

type EchoSource struct {
	logger zerolog.Logger
	source system.Source
}

// NewEchoSource creates a new echo handler struct that has a logger and an underlying source.
func NewEchoSource(logger zerolog.Logger, source system.Source) *EchoSource {
	return &EchoSource{
		logger: logger.With().Str("module", "source-echo-router").Logger(),
		source: source,
	}
}

// Attach takes the incoming existing echo.Echo router, and adds the following routes to it:
// - GET /system/v1/state
// - GET /system/v1/overview
// - GET /system/v1/tenant/:ident
// - GET /system/v1/module/:ident/:ref/:namespace/:mod
// - GET /system/v1/workflows/:ident/:namespace/:version
// - GET /system/v1/connections/:ident/:namespace/:verion
// - GET /system/v1/authentication/:ident/:namespace/:version
// - GET /system/v1/capabilities/:ident/:namespace/:version
// - GET /system/v1/queries/:ident/:namespace/:version
// - GET /system/v1/file/:ident/:version/*filename
func (es *EchoSource) Attach(e *echo.Echo) {
	v1 := e.Group("/system/v1")

	v1.GET("/state", es.StateHandler())
	v1.GET("/overview", es.OverviewHandler())
	v1.GET("/tenant/:ident", es.TenantOverviewHandler())
	v1.GET("/module/:ident/:ref/:namespace/:mod", es.GetModuleHandler())
	v1.GET("/workflows/:ident/:namespace/:version", es.WorkflowsHandler())
	v1.GET("/connections/:ident/:namespace/:version", es.ConnectionsHandler())
	v1.GET("/authentication/:ident/:namespace/:version", es.AuthenticationHandler())
	v1.GET("/capabilities/:ident/:namespace/:version", es.CapabilitiesHandler())
	v1.GET("/queries/:ident/:namespace/:version", es.QueriesHandler())
	v1.GET("/file/:ident/:version/*filename", es.FileHandler())
}

// StateHandler is a handler to fetch the system State.
func (es *EchoSource) StateHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		state, err := es.source.State()
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError).SetInternal(errors.Wrap(err, "es.source.State()"))
		}

		return c.JSON(http.StatusOK, state)
	}
}

// OverviewHandler is a handler to fetch the system overview.
func (es *EchoSource) OverviewHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		overview, err := es.source.Overview()
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError).SetInternal(errors.Wrap(err, "es.source.Overview()"))
		}

		return c.JSON(http.StatusOK, overview)
	}
}

// TenantOverviewHandler is a handler to fetch a particular tenant's overview.
func (es *EchoSource) TenantOverviewHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		ident := c.Param("ident")

		tenantOverview, err := es.source.TenantOverview(ident)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError).SetInternal(errors.Wrap(err, "es.source.TenantOverview()"))
		}

		return c.JSON(http.StatusOK, tenantOverview)
	}
}

// GetModuleHandler is a handler to find a single module.
func (es *EchoSource) GetModuleHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		ident := c.Param("ident")
		ref := c.Param("ref")
		namespace := c.Param("namespace")
		mod := c.Param("mod")

		fqmnString, err := fqmn.FromParts(ident, namespace, mod, ref)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError).SetInternal(errors.Wrap(err, "fqmn.FromParts"))
		}

		module, err := es.source.GetModule(fqmnString)
		if err != nil {
			es.logger.Err(err).Msg("es.source.GetModule")

			if errors.Is(err, system.ErrModuleNotFound) {
				return echo.NewHTTPError(http.StatusNotFound)
			} else if errors.Is(err, system.ErrAuthenticationFailed) {
				return echo.NewHTTPError(http.StatusUnauthorized)
			}

			return echo.NewHTTPError(http.StatusInternalServerError).SetInternal(errors.Wrap(err, "es.source.GetModule"))
		}

		return c.JSON(http.StatusOK, module)
	}
}

// WorkflowsHandler is a handler to fetch Workflows.
func (es *EchoSource) WorkflowsHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		ident := c.Param("ident")
		namespace := c.Param("namespace")
		version, err := strconv.Atoi(c.Param("version"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest).SetInternal(errors.Wrap(err, "strconv.Atoi"))
		}

		workflows, err := es.source.Workflows(ident, namespace, int64(version))
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError).SetInternal(errors.Wrap(err, "es.source.Workflows"))
		}

		return c.JSON(http.StatusOK, workflows)
	}
}

// ConnectionsHandler is a handler to fetch Connection data.
func (es *EchoSource) ConnectionsHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		ident := c.Param("ident")
		namespace := c.Param("namespace")
		version, err := strconv.Atoi(c.Param("version"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest).SetInternal(errors.Wrap(err, "strconv.Atoi"))
		}

		connections, err := es.source.Connections(ident, namespace, int64(version))
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError).SetInternal(errors.Wrap(err, "es.source.Connections"))
		}

		return c.JSON(http.StatusOK, connections)
	}
}

// AuthenticationHandler is a handler to fetch Authentication data.
func (es *EchoSource) AuthenticationHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		ident := c.Param("ident")
		namespace := c.Param("namespace")
		version, err := strconv.Atoi(c.Param("version"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest).SetInternal(errors.Wrap(err, "strconv.Atoi"))
		}

		authentication, err := es.source.Authentication(ident, namespace, int64(version))
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError).SetInternal(errors.Wrap(err, "es.source.Authentication"))
		}

		return c.JSON(http.StatusOK, authentication)
	}
}

// CapabilitiesHandler is a handler to fetch Capabilities data.
func (es *EchoSource) CapabilitiesHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		ident := c.Param("ident")
		namespace := c.Param("namespace")
		version, err := strconv.Atoi(c.Param("version"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest).SetInternal(errors.Wrap(err, "strconv.Atoi"))
		}

		caps, err := es.source.Capabilities(ident, namespace, int64(version))
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError).SetInternal(errors.Wrap(err, "es.source.Capabilities"))
		}

		return c.JSON(http.StatusOK, caps)
	}
}

// FileHandler is a handler to fetch Files.
func (es *EchoSource) FileHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		ident := c.Param("ident")
		filename := c.Param("filename")

		version, err := strconv.Atoi(c.Param("version"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest).SetInternal(errors.Wrap(err, "strconv.Atoi"))
		}

		fileBytes, err := es.source.StaticFile(ident, int64(version), filename)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return echo.NewHTTPError(http.StatusNotFound)
			}

			return echo.NewHTTPError(http.StatusInternalServerError).SetInternal(errors.Wrap(err, "es.source.StaticFile"))
		}

		return c.Blob(http.StatusOK, "text/plain", fileBytes)
	}
}

// QueriesHandler is a handler to fetch queries.
func (es *EchoSource) QueriesHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		ident := c.Param("ident")
		namespace := c.Param("namespace")
		version, err := strconv.Atoi(c.Param("version"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest).SetInternal(errors.Wrap(err, "strconv.Atoi"))
		}

		queries, err := es.source.Queries(ident, namespace, int64(version))
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError).SetInternal(errors.Wrap(err, "es.source.Queries"))
		}

		return c.JSON(http.StatusOK, queries)
	}
}
