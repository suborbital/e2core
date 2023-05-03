package sourceserver

import (
	"fmt"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/suborbital/go-kit/web/mid"
	"github.com/suborbital/systemspec/system/bundle"
)

func FromBundle(bundlePath string) (*echo.Echo, error) {
	bs := bundle.NewBundleSource(bundlePath)

	if err := bs.Start(); err != nil {
		return nil, errors.Wrap(err, "failed to Start bundle source")
	}

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	l := zerolog.New(os.Stderr).With().
		Timestamp().
		Str("service", "systemSourceServer").
		Logger()

	e := echo.New()

	e.Use(
		mid.UUIDRequestID(),
		mid.Logger(l, nil),
		middleware.Recover(),
	)

	rt := NewRouter(l, bs)

	if err := rt.Attach("/system/v1", e); err != nil {
		return nil, errors.Wrap(err, "es.Attach with /system/v1 prefix")
	}

	return e, nil
}

// Start starts the given sourceserver and returns any errors
func Start(e *echo.Echo) error {
	if e == nil {
		return nil
	}

	if err := e.Start(fmt.Sprintf(":%d", 9090)); err != nil {
		return errors.Wrap(err, "failed to e.Start")
	}

	return nil
}
