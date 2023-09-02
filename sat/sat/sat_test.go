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
	sdkTrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/suborbital/e2core/foundation/tracing"
	"github.com/suborbital/e2core/sat/sat/metrics"
	"github.com/suborbital/e2core/sat/sat/options"
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

	assert.Equal(t, `{"message":"don't go there","status":401}
`, string(body))
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

	assert.Equal(t, `{"message":"unknown error","status":500}
`, string(body))
}

func satForFile(filepath string) (*Sat, *sdkTrace.TracerProvider, error) {
	config, err := ConfigFromModuleArg(zerolog.Nop(), options.Options{}, filepath)
	if err != nil {
		return nil, nil, err
	}

	traceProvider, err := tracing.SetupTracing(tracing.Config{}, zerolog.Nop())
	if err != nil {
		return nil, nil, errors.Wrap(err, "setup tracing")
	}

	sat, err := New(config, zerolog.Nop(), metrics.SetupNoopMetrics())
	if err != nil {
		return nil, nil, err
	}

	return sat, traceProvider, nil
}
