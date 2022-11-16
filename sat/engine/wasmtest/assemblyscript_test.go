package wasmtest

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/suborbital/e2core/sat/api"
	"github.com/suborbital/e2core/sat/engine"
	"github.com/suborbital/systemspec/capabilities"
	"github.com/suborbital/systemspec/request"
)

func TestASEcho(t *testing.T) {
	e := engine.New()

	// test a WASM module that is loaded directly instead of through the bundle
	doWasm, err := e.RegisterFromFile("as-echo", "../testdata/as-echo/as-echo.wasm")
	require.NoError(t, err, "registerfrom file failed for as-echo.wasm")

	res, err := doWasm("from AssemblyScript!").Then()
	if err != nil {
		t.Error(errors.Wrap(err, "failed to Then"))
		return
	}

	fmt.Println(string(res.([]byte)))

	if string(res.([]byte)) != "hello, from AssemblyScript!" {
		t.Error("as-echo failed, got:", string(res.([]byte)))
	}
}

func TestASFetch(t *testing.T) {
	e := engine.New()

	// test a WASM module that is loaded directly instead of through the bundle
	doWasm, err := e.RegisterFromFile("as-fetch", "../testdata/as-fetch/as-fetch.wasm")
	require.NoError(t, err, "registerfrom file failed for as-fetch.wasm")

	res, err := doWasm("https://1password.com").Then()
	if err != nil {
		t.Error(errors.Wrap(err, "failed to Then"))
		return
	}

	if string(res.([]byte)[:100]) != "<!doctype html><html lang=en data-language-url=/><head><meta charset=utf-8><meta name=viewport conte" {
		t.Error("as-fetch failed, got:", string(res.([]byte)[:100]))
	}
}

func TestASGraphql(t *testing.T) {
	// bail out if GitHub auth is not set up (i.e. in Travis)
	if _, ok := os.LookupEnv("GITHUB_TOKEN"); !ok {
		return
	}

	config := capabilities.DefaultCapabilityConfig()
	config.Auth = &capabilities.AuthConfig{
		Enabled: true,
		Headers: map[string]capabilities.AuthHeader{
			"api.github.com": {
				HeaderType: "bearer",
				Value:      "env(GITHUB_TOKEN)",
			},
		},
	}

	api, _ := api.NewWithConfig(config)

	e := engine.NewWithAPI(api)

	// test a WASM module that is loaded directly instead of through the bundle
	doWasm, err := e.RegisterFromFile("as-graphql", "../testdata/as-graphql/as-graphql.wasm")
	require.NoError(t, err, "registerfrom file failed for as-graphql.wasm")

	res, err := doWasm(nil).Then()
	if err != nil {
		t.Error(errors.Wrap(err, "failed to Then"))
		return
	}

	if string(res.([]byte)) != `{"data":{"repository":{"name":"reactr","nameWithOwner":"suborbital/reactr"}}}` {
		t.Error("as-graphql failed, got:", string(res.([]byte)))
	}
}

func TestASLargeData(t *testing.T) {
	e := engine.New()

	// test a WASM module that is loaded directly instead of through the bundle
	doWasm, err := e.RegisterFromFile("as-echo", "../testdata/as-echo/as-echo.wasm")
	require.NoError(t, err, "registerfrom file failed for as-echo.wasm")

	res, err := doWasm(largeInput).Then()
	if err != nil {
		t.Error(errors.Wrap(err, "failed to Then"))
		return
	}

	if string(res.([]byte)) != "hello, "+largeInput {
		t.Error("as-test failed, got:", string(res.([]byte)))
	}
}

func TestASRunnerWithRequest(t *testing.T) {
	e := engine.New()

	doWasm, err := e.RegisterFromFile("wasm", "../testdata/as-req/as-req.wasm")
	require.NoError(t, err, "registerfrom file failed for as-req.wasm")

	body := testBody{
		Username: "cohix",
	}

	bodyJSON, _ := json.Marshal(body)

	req := &request.CoordinatedRequest{
		Method: "GET",
		URL:    "/hello/world",
		ID:     uuid.New().String(),
		Body:   bodyJSON,
		State: map[string][]byte{
			"hello": []byte("what is up"),
		},
	}

	reqJSON, err := req.ToJSON()
	if err != nil {
		t.Error("failed to ToJSON", err)
	}

	res, err := doWasm(reqJSON).Then()
	if err != nil {
		t.Error(errors.Wrap(err, "failed to Then"))
		return
	}

	resp := res.(*request.CoordinatedResponse)

	if string(resp.Output) != "hello what is up" {
		t.Error(fmt.Errorf("expected 'hello, what is up', got %s", string(res.([]byte))))
	}
}
