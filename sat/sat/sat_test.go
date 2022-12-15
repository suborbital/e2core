package sat

import (
	"bytes"
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/trace"

	"github.com/suborbital/e2core/sat/sat/metrics"
	"github.com/suborbital/vektor/vtest"
)

func TestEchoRequest(t *testing.T) {
	sat, tp, err := satForFile("../examples/hello-echo/hello-echo.wasm")
	require.NoError(t, err)

	ctx, ctxCloser := context.WithTimeout(context.Background(), time.Second)
	defer ctxCloser()
	defer tp.Shutdown(ctx)

	vt := vtest.New(sat.testServer())

	req, _ := http.NewRequest(http.MethodPost, "/", bytes.NewBuffer([]byte("my friend")))

	resp := vt.Do(req, t)

	resp.AssertStatus(200)
	resp.AssertBodyString("hello my friend")
}

func TestEchoGetRequest(t *testing.T) {
	sat, tp, err := satForFile("../examples/hello-echo/hello-echo.wasm")
	require.NoError(t, err)

	ctx, ctxCloser := context.WithTimeout(context.Background(), time.Second)
	defer ctxCloser()
	defer tp.Shutdown(ctx)

	vt := vtest.New(sat.testServer())

	req, _ := http.NewRequest(http.MethodGet, "/", bytes.NewBuffer(nil))

	resp := vt.Do(req, t)

	resp.AssertStatus(200)
}

func TestErrorRequest(t *testing.T) {
	sat, tp, err := satForFile("../examples/return-err/return-err.wasm")
	require.NoError(t, err)

	ctx, ctxCloser := context.WithTimeout(context.Background(), time.Second)
	defer ctxCloser()
	defer tp.Shutdown(ctx)

	vt := vtest.New(sat.testServer())

	req, _ := http.NewRequest(http.MethodPost, "/", bytes.NewBuffer([]byte{}))

	resp := vt.Do(req, t)

	resp.AssertStatus(401)
	resp.AssertBodyString(`{"status":401,"message":"don't go there"}`)
}

func TestPanicRequest(t *testing.T) {
	sat, tp, err := satForFile("../examples/panic-at-the-disco/panic-at-the-disco.wasm")
	require.NoError(t, err)

	ctx, ctxCloser := context.WithTimeout(context.Background(), time.Second)
	defer ctxCloser()
	defer tp.Shutdown(ctx)

	vt := vtest.New(sat.testServer())

	req, _ := http.NewRequest(http.MethodPost, "/", bytes.NewBuffer([]byte{}))

	resp := vt.Do(req, t)

	resp.AssertStatus(500)
	resp.AssertBodyString(`{"status":500,"message":"unknown error"}`)
}

func satForFile(filepath string) (*Sat, *trace.TracerProvider, error) {
	config, err := ConfigFromModuleArg(filepath)
	if err != nil {
		return nil, nil, err
	}

	traceProvider, err := SetupTracing(config.TracerConfig, config.Logger)
	if err != nil {
		return nil, nil, errors.Wrap(err, "setup tracing")
	}

	sat, err := New(config, traceProvider, metrics.SetupNoopMetrics())
	if err != nil {
		return nil, nil, err
	}

	return sat, traceProvider, nil
}
