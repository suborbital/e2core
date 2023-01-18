package enginetest

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/suborbital/e2core/foundation/scheduler"
	"github.com/suborbital/e2core/sat/engine2"
	"github.com/suborbital/e2core/sat/engine2/api"
	"github.com/suborbital/systemspec/request"
	"github.com/suborbital/vektor/vlog"
)

type testBody struct {
	Username string `json:"username"`
}

func TestEngineWithRequest(t *testing.T) {
	ref, err := engine2.WasmRefFromFile("./testdata/log/log.wasm")
	if err != nil {
		t.Error(err)
		return
	}

	e := engine2.New("log", ref, api.New(vlog.Default()))

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

	res, err := e.Do(scheduler.NewJob("log", req)).Then()
	if err != nil {
		t.Error(errors.Wrap(err, "failed to Then"))
		return
	}

	resp := res.(*request.CoordinatedResponse)

	if string(resp.Output) != "hello what is up" {
		t.Error(fmt.Errorf("expected 'hello, what is up', got %s", string(res.([]byte))))
	}
}

func TestEngineWithURLQuery(t *testing.T) {
	ref, err := engine2.WasmRefFromFile("./testdata/rust-urlquery/rust-urlquery.wasm")
	if err != nil {
		t.Error(err)
		return
	}

	e := engine2.New("rust-urlquery", ref, api.New(vlog.Default()))

	req := &request.CoordinatedRequest{
		Method: "GET",
		URL:    "/hello/world?message=whatsup",
		ID:     uuid.New().String(),
		Body:   []byte{},
	}

	res, err := e.Do(scheduler.NewJob("rust-urlquery", req)).Then()
	if err != nil {
		t.Error(errors.Wrap(err, "failed to Then"))
		return
	}

	resp := res.(*request.CoordinatedResponse)

	if string(resp.Output) != "hello whatsup" {
		t.Error(fmt.Errorf("expected 'hello, whatsup', got %s", string(res.([]byte))))
	}
}

func TestEngineSetRespHeader(t *testing.T) {
	ref, err := engine2.WasmRefFromFile("./testdata/rs-reqset/rs-reqset.wasm")
	if err != nil {
		t.Error(err)
		return
	}

	e := engine2.New("rs-reqset", ref, api.New(vlog.Default()))

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
		Headers: map[string]string{},
	}

	_, err = e.Do(scheduler.NewJob("rs-reqset", req)).Then()
	if err != nil {
		t.Error(errors.Wrap(err, "failed to Then"))
		return
	}

	if val, ok := req.Headers["X-REACTR-TEST"]; !ok {
		t.Error("header was not set correctly")
	} else if val != "test successful!" {
		t.Error(fmt.Errorf("expected 'test successful!', got %s", val))
	}
}

func TestEngineFetch(t *testing.T) {
	ref, err := engine2.WasmRefFromFile("./testdata/fetch/fetch.wasm")
	if err != nil {
		t.Error(err)
		return
	}

	e := engine2.New("fetch", ref, api.New(vlog.Default()))

	req := &request.CoordinatedRequest{
		Method: "GET",
		URL:    "/hello/world",
		ID:     uuid.New().String(),
		Body:   []byte("https://1password.com"),
	}

	res, err := e.Do(scheduler.NewJob("fetch", req)).Then()
	if err != nil {
		t.Error(errors.Wrap(err, "failed to Then"))
		return
	}

	resp := res.(*request.CoordinatedResponse)

	if len(resp.Output) < 100 {
		t.Errorf("expected 1password.com HTML, got %q", string(res.([]byte)))
	}

	fmt.Println(string(resp.Output[:1000]))

	if string(resp.Output)[:1000] != "<!doctype html><html lang=en data-language-url=/><head><meta charset=utf-8><meta name=viewport content=\"width=device-width,initial-scale=1,maximum-scale=5,user-scalable=yes\"><meta http-equiv=x-ua-compatible content=\"IE=edge, chrome=1\"><meta name=theme-color content=\"#1a8cff\"><link rel=alternate hreflang=x-default href=https://1password.com/><link rel=alternate hreflang=en-us href=https://1password.com/><link rel=alternate hreflang=en-ca href=https://1password.com/><link rel=alternate hreflang=en-au href=https://1password.com/><link rel=alternate hreflang=en-nz href=https://1password.com/><link rel=alternate hreflang=en-in href=https://1password.com/><link rel=alternate hreflang=pt-br href=https://1password.com/pt/><link rel=alternate hreflang=es-es href=https://1password.com/es/><link rel=alternate hreflang=it-it href=https://1password.com/it/><link rel=alternate hreflang=fr-fr href=https://1password.com/fr/><link rel=alternate hreflang=de-de href=https://1password.com/de/><link rel=al" {
		t.Errorf("expected 1password homepage response, got %q", string(res.([]byte))[:1000])
	}
}
