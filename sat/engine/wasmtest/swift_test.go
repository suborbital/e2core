package wasmtest

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/suborbital/e2core/sat/api"
	"github.com/suborbital/e2core/sat/engine"
	"github.com/suborbital/e2core/scheduler"
	"github.com/suborbital/systemspec/capabilities"
	"github.com/suborbital/systemspec/request"
)

func TestWasmRunnerWithFetchSwift(t *testing.T) {
	e := engine.New()

	e.RegisterFromFile("fetch-swift", "../testdata/fetch-swift/fetch-swift.wasm")

	job := scheduler.NewJob("fetch-swift", "https://1password.com")

	res, err := e.Do(job).Then()
	if err != nil {
		t.Error(errors.Wrap(err, "failed to Then"))
		return
	}

	if string(res.([]byte))[:100] != "<!doctype html><html lang=en data-language-url=/><head><meta charset=utf-8><meta name=viewport conte" {
		t.Error(fmt.Errorf("expected 1password.com HTML, got %q", string(res.([]byte))[:100]))
	}
}

func TestWasmRunnerEchoSwift(t *testing.T) {
	e := engine.New()

	e.RegisterFromFile("hello-swift", "../testdata/hello-swift/hello-swift.wasm")

	job := scheduler.NewJob("hello-swift", "Connor")

	res, err := e.Do(job).Then()
	if err != nil {
		t.Error(errors.Wrap(err, "failed to Then"))
		return
	}

	if string(res.([]byte)) != "hello Connor" {
		t.Error(fmt.Errorf("hello Connor, got %s", string(res.([]byte))))
	}
}

func TestWasmRunnerSwift(t *testing.T) {
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

	e := engine.New()

	e.RegisterFromFile("swift-log", "../testdata/swift-log/swift-log.wasm")

	job := scheduler.NewJob("swift-log", reqJSON)

	res, err := e.Do(job).Then()
	if err != nil {
		t.Error(errors.Wrap(err, "failed to Then"))
		return
	}

	resp := res.(*request.CoordinatedResponse)

	if string(resp.Output) != "hello what is up" {
		t.Error(fmt.Errorf("expected 'hello, what is up', got %s", string(res.([]byte))))
	}
}

func TestWasmFileGetStaticSwift(t *testing.T) {
	config := capabilities.DefaultCapabilityConfig()
	config.File = fileConfig

	api, _ := api.NewWithConfig(config)

	e := engine.NewWithAPI(api)

	e.RegisterFromFile("get-static-swift", "../testdata/get-static-swift/get-static-swift.wasm")

	getJob := scheduler.NewJob("get-static-swift", "")

	res, err := e.Do(getJob).Then()
	if err != nil {
		t.Error(errors.Wrap(err, "failed to Do get-static job"))
		return
	}

	result := string(res.([]byte))

	expected := "# Hello, World\n\nContents are very important"

	if result != expected {
		t.Error("failed, got:\n", result, "\nexpeted:\n", expected)
	}
}
