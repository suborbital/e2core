package sat

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/trace"

	"github.com/suborbital/e2core/sat/sat/metrics"
)

func TestEchoRequest(t *testing.T) {
	sat, tp, err := satForFile("../examples/hello-echo/hello-echo.wasm")
	require.NoError(t, err)

	ctx, ctxCloser := context.WithTimeout(context.Background(), time.Second)
	defer ctxCloser()
	defer tp.Shutdown(ctx)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer([]byte("my friend")))
	w := httptest.NewRecorder()

	sat.server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	body, err := io.ReadAll(w.Result().Body)
	require.NoError(t, err)

	assert.Equal(t, "hello my friend", string(body))
}

func TestEchoGetRequest(t *testing.T) {
	sat, tp, err := satForFile("../examples/hello-echo/hello-echo.wasm")
	require.NoError(t, err)

	ctx, ctxCloser := context.WithTimeout(context.Background(), time.Second)
	defer ctxCloser()
	defer tp.Shutdown(ctx)

	req := httptest.NewRequest(http.MethodGet, "/", bytes.NewBuffer(nil))
	w := httptest.NewRecorder()

	sat.server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
}

func TestErrorRequest(t *testing.T) {
	sat, tp, err := satForFile("../examples/return-err/return-err.wasm")
	require.NoError(t, err)

	ctx, ctxCloser := context.WithTimeout(context.Background(), time.Second)
	defer ctxCloser()
	defer tp.Shutdown(ctx)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer([]byte{}))
	w := httptest.NewRecorder()

	sat.server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Result().StatusCode)
	body, err := io.ReadAll(w.Result().Body)
	require.NoError(t, err)

	assert.Equal(t, `{"status":401,"message":"don't go there"}`, string(body))
}

func TestPanicRequest(t *testing.T) {
	sat, tp, err := satForFile("../examples/panic-at-the-disco/panic-at-the-disco.wasm")
	require.NoError(t, err)

	ctx, ctxCloser := context.WithTimeout(context.Background(), time.Second)
	defer ctxCloser()
	defer tp.Shutdown(ctx)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer([]byte{}))
	w := httptest.NewRecorder()

	sat.server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
	body, err := io.ReadAll(w.Result().Body)
	require.NoError(t, err)

	assert.Equal(t, `{"status":500,"message":"unknown error"}`, string(body))
}

func satForFile(filepath string) (*Sat, *trace.TracerProvider, error) {
	config, err := ConfigFromModuleArg(zerolog.Nop(), filepath)
	if err != nil {
		return nil, nil, err
	}

	traceProvider, err := SetupTracing(config.TracerConfig, zerolog.Nop())
	if err != nil {
		return nil, nil, errors.Wrap(err, "setup tracing")
	}

	sat, err := New(config, zerolog.Nop(), traceProvider, metrics.SetupNoopMetrics())
	if err != nil {
		return nil, nil, err
	}

	return sat, traceProvider, nil
}
