package wasmtest

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/suborbital/e2core/sat/api"
	"github.com/suborbital/e2core/sat/engine"
	"github.com/suborbital/e2core/scheduler"
	"github.com/suborbital/systemspec/capabilities"
	"github.com/suborbital/systemspec/request"
)

func TestGrainEcho(t *testing.T) {
	e := engine.New()

	// test a WASM module that is loaded directly instead of through the bundle
	doWasm, _ := e.RegisterFromFile("grain-echo", "../testdata/grain-echo/grain-echo.wasm")

	res, err := doWasm("from Grain!").Then()
	if err != nil {
		t.Error(errors.Wrap(err, "failed to Then"))
		return
	}

	fmt.Println(string(res.([]byte)))

	if string(res.([]byte)) != "hello, from Grain!" {
		t.Error("grain-echo failed, got:", string(res.([]byte)))
	}
}

func TestGrainFetch(t *testing.T) {
	e := engine.New()

	// test a WASM module that is loaded directly instead of through the bundle
	doWasm, _ := e.RegisterFromFile("grain-fetch", "../testdata/grain-fetch/grain-fetch.wasm")

	res, err := doWasm("https://1password.com").Then()
	if err != nil {
		t.Error(errors.Wrap(err, "failed to Then"))
		return
	}

	if string(res.([]byte)[:100]) != "<!doctype html><html lang=en data-language-url=/><head><meta charset=utf-8><meta name=viewport conte" {
		t.Error("grain-fetch failed, got:", string(res.([]byte)[:100]))
	}
}

func TestGrainGraphql(t *testing.T) {
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
	e.RegisterFromFile("grain-graphql", "../testdata/grain-graphql/grain-graphql.wasm")

	res, err := e.Do(scheduler.NewJob("grain-graphql", nil)).Then()
	if err != nil {
		t.Error(errors.Wrap(err, "failed to Then"))
		return
	}

	if string(res.([]byte)) != `{"data":{"repository":{"name":"reactr","nameWithOwner":"suborbital/reactr"}}}` {
		t.Error("grain-graphql failed, got:", string(res.([]byte)))
	}
}

func TestGrainLargeData(t *testing.T) {
	e := engine.New()

	// test a WASM module that is loaded directly instead of through the bundle
	doWasm, _ := e.RegisterFromFile("grain-echo", "../testdata/grain-echo/grain-echo.wasm")

	res, err := doWasm(largeInput).Then()
	if err != nil {
		t.Error(errors.Wrap(err, "failed to Then"))
		return
	}

	if string(res.([]byte)) != "hello, "+largeInput {
		t.Error("grain-test failed, got:", string(res.([]byte)))
	}
}

func TestGrainRunnerWithRequest(t *testing.T) {
	e := engine.New()

	doWasm, _ := e.RegisterFromFile("wasm", "../testdata/grain-req/grain-req.wasm")

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
