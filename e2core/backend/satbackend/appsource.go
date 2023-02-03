package satbackend

import (
	"fmt"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	e2Server "github.com/suborbital/e2core/e2core/server"
	"github.com/suborbital/go-kit/web/mid"
	"github.com/suborbital/systemspec/system/bundle"
)

func startSystemSourceServer(bundlePath string) (chan error, error) {
	app := bundle.NewBundleSource(bundlePath)

	l := zerolog.New(os.Stderr).With().Str("service", "systemSourceServer").Timestamp().Logger()

	e := echo.New()
	e.Use(
		mid.UUIDRequestID(),
		mid.Logger(l, nil),
	)

	es := e2Server.NewEchoSource(l, app)
	es.Attach(e)

	errChan := make(chan error)

	go func() {
		if err := e.Start(fmt.Sprintf(":%d", 9090)); err != nil {
			errChan <- errors.Wrap(err, "failed to server.Start")
		}
	}()

	return errChan, nil
}
