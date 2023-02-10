package enginetest

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/suborbital/e2core/foundation/scheduler"
	"github.com/suborbital/e2core/sat/engine2"
	"github.com/suborbital/e2core/sat/engine2/api"
	"github.com/suborbital/systemspec/request"
)

func TestEngineWithTinyGo(t *testing.T) {
	ref, err := engine2.WasmRefFromFile("./testdata/tinygo-urlquery/tinygo-urlquery.wasm")
	if err != nil {
		t.Error(err)
		return
	}

	e := engine2.New("tinygo-urlquery", ref, api.New(zerolog.Nop()))

	req := &request.CoordinatedRequest{
		Method: "GET",
		URL:    "/hello/world?message=whatsup",
		ID:     uuid.New().String(),
		Body:   []byte{},
	}

	res, err := e.Do(scheduler.NewJob("tinygo-urlquery", req)).Then()
	if err != nil {
		t.Error(errors.Wrap(err, "failed to Then"))
		return
	}

	resp := res.(*request.CoordinatedResponse)

	if string(resp.Output) != "hello whatsup" {
		t.Error(fmt.Errorf("expected 'hello whatsup', got %s", string(resp.Output)))
	}
}

func TestEngineTinyGoUrlQuery(t *testing.T) {
	ref, err := engine2.WasmRefFromFile("./testdata/tinygo-resp/tinygo-resp.wasm")
	if err != nil {
		t.Error(err)
		return
	}

	e := engine2.New("tinygo-resp", ref, api.New(zerolog.Nop()))

	req := &request.CoordinatedRequest{
		Method: "POST",
		URL:    "/hello/world",
		ID:     uuid.New().String(),
		Body:   []byte("world"),
	}

	res, err := e.Do(scheduler.NewJob("tinygo-resp", req)).Then()
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
