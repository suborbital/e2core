package satbackend

import (
	"net/http"
	"strconv"
	"strings"

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

// Routes creates an echo route group and returns it so the consuming service can attach it to wherever it likes using
// whatever middlewares they way. The consuming service no longer has an option to modify the middlewares of a route,
// nor can it remove a route once added. The routes do not have a prefix, so the consuming service can choose what path
// to mount these under.
//
// To use this you will need to do the following:
// - e.Any("/some/prefix", echo.WrapHandler(es.Routes(), <middlewares>)
//
// Alternatively you can not use the Routes() method, and construct your own route-handler pairs as you see fit, because
// all of the handlers are exported. That way you have the most flexibility.
//
// The routes in this group are the following:
// - GET /state
// - GET /overview
// - GET /tenant/:ident
// - GET /module/:ident/:ref/:namespace/:mod
// - GET /workflows/:ident/:namespace/:version
// - GET /connections/:ident/:namespace/:verion
// - GET /authentication/:ident/:namespace/:version
// - GET /capabilities/:ident/:namespace/:version
// - GET /queries/:ident/:namespace/:version
// - GET /file/:ident/:version/*filename
func (es *EchoSource) Routes() *echo.Echo {
	e := echo.New()
	v1 := e.Group("/")

	v1.GET("/state", es.StateHandler())
	v1.GET("/overview", es.OverviewHandler())
	v1.GET("/tenant/:ident", es.TenantOverviewHandler())
	v1.GET("/module/:ident/:ref/:namespace/:mod", es.GetModuleHandler())
	v1.GET("/workflows/:ident/:namespace/:version", es.WorkflowsHandler())
	v1.GET("/connections/:ident/:namespace/:version", es.ConnectionsHandler())
	v1.GET("/authentication/:ident/:namespace/:version", es.AuthenticationHandler())
	v1.GET("/capabilities/:ident/:namespace/:version", es.CapabilitiesHandler())

	return e
}

// Attach takes a prefix and an echo instance to attach the routes onto. The prefix can either be empty, or start with
// a / character. It will attach the following routes to the passed in echo handler:
// - GET /<prefix>/state
// - GET /<prefix>/overview
// - GET /<prefix>/tenant/:ident
// - GET /<prefix>/module/:ident/:ref/:namespace/:mod
// - GET /<prefix>/workflows/:ident/:namespace/:version
// - GET /<prefix>/connections/:ident/:namespace/:verion
// - GET /<prefix>/authentication/:ident/:namespace/:version
// - GET /<prefix>/capabilities/:ident/:namespace/:version
// - GET /<prefix>/queries/:ident/:namespace/:version
// - GET /<prefix>/file/:ident/:version/*filename
//
// If the prefix is not empty and does not start with a / character, it returns an error.
func (es *EchoSource) Attach(prefix string, e *echo.Echo) error {
	if prefix == "" {
		prefix = "/"
	}

	if !strings.HasPrefix(prefix, "/") {
		return errors.New("prefix must start with a / character")
	}

	v1 := e.Group(prefix)
	v1.GET("/state", es.StateHandler())
	v1.GET("/overview", es.OverviewHandler())
	v1.GET("/tenant/:ident", es.TenantOverviewHandler())
	v1.GET("/module/:ident/:ref/:namespace/:mod", es.GetModuleHandler())
	v1.GET("/workflows/:ident/:namespace/:version", es.WorkflowsHandler())
	v1.GET("/connections/:ident/:namespace/:version", es.ConnectionsHandler())
	v1.GET("/authentication/:ident/:namespace/:version", es.AuthenticationHandler())
	v1.GET("/capabilities/:ident/:namespace/:version", es.CapabilitiesHandler())

	return nil
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
