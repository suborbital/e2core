package satbackend

import (
	"fmt"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/suborbital/go-kit/web/mid"
	"github.com/suborbital/systemspec/system/bundle"
)

func startSystemSourceServer(bundlePath string) (chan error, error) {
	app := bundle.NewBundleSource(bundlePath)

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	l := zerolog.New(os.Stderr).With().
		Timestamp().
		Str("service", "systemSourceServer").
		Logger()

	e := echo.New()
	e.Use(
		mid.UUIDRequestID(),
		mid.Logger(l, nil),
	)

	es := NewEchoSource(l, app)
	err := es.Attach("/system/v1", e)
	if err != nil {
		return nil, errors.Wrap(err, "es.Attach with /system/v1 prefix")
	}

	errChan := make(chan error)

	go func() {
		if err := e.Start(fmt.Sprintf(":%d", 9090)); err != nil {
			errChan <- errors.Wrap(err, "failed to server.Start")
		}
	}()

	return errChan, nil
}
