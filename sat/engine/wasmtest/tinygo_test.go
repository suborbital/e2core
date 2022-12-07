package wasmtest

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/suborbital/e2core/foundation/scheduler"
	"github.com/suborbital/e2core/sat/engine"
	"github.com/suborbital/e2core/sat/engine/runtime/api"
	"github.com/suborbital/systemspec/capabilities"
	"github.com/suborbital/systemspec/request"
)

func TestWasmRunnerTinyGo(t *testing.T) {
	e := engine.New()

	// test a WASM module that is loaded directly instead of through the bundle
	doWasm, _ := e.RegisterFromFile("wasm", "../testdata/tinygo-hello-echo/tinygo-hello-echo.wasm")

	res, err := doWasm("world").Then()
	if err != nil {
		t.Error(errors.Wrap(err, "failed to Then"))
		return
	}

	if string(res.([]byte)) != "Hello, world" {
		t.Errorf("expected Hello world got %q", string(res.([]byte)))
	}
}

func TestWasmFileGetStaticTinyGo(t *testing.T) {
	config := capabilities.DefaultCapabilityConfig()
	config.File = fileConfig

	api, _ := api.NewWithConfig(config)

	e := engine.NewWithAPI(api)

	e.RegisterFromFile("tinygo-get-static", "../testdata/tinygo-get-static/tinygo-get-static.wasm")

	getJob := scheduler.NewJob("tinygo-get-static", "")

	res, err := e.Do(getJob).Then()
	if err != nil {
		t.Error(errors.Wrap(err, "failed to Do tinygo-get-static job"))
		return
	}

	result := string(res.([]byte))

	expected := "# Hello, World\n\nContents are very important"

	if result != expected {
		t.Error("failed, got:\n", result, "\nexpected:\n", expected)
	}
}

func TestGoURLQuery(t *testing.T) {
	e := engine.New()

	// using a Rust module
	doWasm, _ := e.RegisterFromFile("wasm", "../testdata/tinygo-urlquery/tinygo-urlquery.wasm")

	req := &request.CoordinatedRequest{
		Method: "GET",
		URL:    "/hello/world?message=whatsup",
		ID:     uuid.New().String(),
		Body:   []byte{},
	}

	res, err := doWasm(req).Then()
	if err != nil {
		t.Error(errors.Wrap(err, "failed to Then"))
		return
	}

	resp := res.(*request.CoordinatedResponse)

	if string(resp.Output) != "hello whatsup" {
		t.Error(fmt.Errorf("expected 'hello whatsup', got %s", string(resp.Output)))
	}
}

func TestGoContentType(t *testing.T) {
	req := &request.CoordinatedRequest{
		Method: "POST",
		URL:    "/hello/world",
		ID:     uuid.New().String(),
		Body:   []byte("world"),
	}

	reqJSON, err := req.ToJSON()
	if err != nil {
		t.Error("failed to ToJSON", err)
	}

	e := engine.New()

	e.RegisterFromFile("content-type", "../testdata/tinygo-resp/tinygo-resp.wasm")

	job := scheduler.NewJob("content-type", reqJSON)

	res, err := e.Do(job).Then()
	if err != nil {
		t.Error(errors.Wrap(err, "failed to Then"))
		return
	}

	resp := res.(*request.CoordinatedResponse)

	if resp.RespHeaders["Content-Type"] != "application/json" {
		t.Error(fmt.Errorf("expected 'Content-Type: application/json', got %s", resp.RespHeaders["Content-Type"]))
	}

	if resp.RespHeaders["X-Reactr"] != string(req.Body) {
		t.Error(fmt.Errorf("expected 'X-Reactr: %s', got %s", string(req.Body), string(req.Body)))
	}
}
