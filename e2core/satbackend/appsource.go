package satbackend

import (
	"github.com/pkg/errors"

	"github.com/suborbital/appspec/system/bundle"
	"github.com/suborbital/appspec/system/server"
	"github.com/suborbital/e2core/options"
	"github.com/suborbital/vektor/vk"
	"github.com/suborbital/vektor/vlog"
)

func startAppSourceServer(bundlePath string) chan error {
	app := bundle.NewBundleSource(bundlePath)
	opts := options.NewWithModifiers()

	errChan := make(chan error)

	router, err := server.NewAppSourceVKRouter(app, opts).GenerateRouter()
	if err != nil {
		errChan <- errors.Wrap(err, "failed to NewAppSourceVKRouter.GenerateRouter")
	}

	log := vlog.Default(
		vlog.Level(vlog.LogLevelWarn),
	)

	server := vk.New(
		vk.UseLogger(log),
		vk.UseAppName("AppSource server"),
		vk.UseHTTPPort(9090),
	)

	server.SwapRouter(router)

	go func() {
		if err := server.Start(); err != nil {
			errChan <- errors.Wrap(err, "failed to server.Start")
		}
	}()

	return errChan
}
