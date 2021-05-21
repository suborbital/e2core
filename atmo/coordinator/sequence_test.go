package coordinator

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/suborbital/atmo/atmo/appsource"
	"github.com/suborbital/atmo/atmo/options"
	"github.com/suborbital/atmo/directive"
	"github.com/suborbital/reactr/request"
	"github.com/suborbital/vektor/vlog"
)

var coord *Coordinator

func init() {
	opts := options.NewWithModifiers(
		options.UseLogger(vlog.Default(
			vlog.Level(vlog.LogLevelDebug),
		)),
	)

	appSource := appsource.NewBundleSource("../../example-project/runnables.wasm.zip")

	coord = New(appSource, opts)

	for {
		if coord.App.Ready() {
			break
		} else {
			time.Sleep(time.Millisecond * 500)
		}
	}
}

func TestBasicSequence(t *testing.T) {
	steps := []directive.Executable{
		{
			CallableFn: directive.CallableFn{
				Fn: "helloworld-rs",
			},
		},
	}

	seq := newSequence(steps, coord.grav.Connect, coord.log)

	req := &request.CoordinatedRequest{
		Method: "GET",
		URL:    "/hello/world",
		ID:     uuid.New().String(),
		Body:   []byte("world"),
		State:  map[string][]byte{},
	}

	state, err := seq.exec(req)
	if err != nil {
		t.Error(err)
	}

	if val, ok := state.state["helloworld-rs"]; !ok {
		t.Error("helloworld state is missing")
	} else if !bytes.Equal(val, []byte("hello world")) {
		t.Error("unexpected helloworld state value:", string(val))
	}
}

func TestGroupSequence(t *testing.T) {
	steps := []directive.Executable{
		{
			Group: []directive.CallableFn{
				{
					Fn: "helloworld-rs",
				},
				{
					Fn: "get-file",
					As: "main.md",
				},
			},
		},
	}

	seq := newSequence(steps, coord.grav.Connect, coord.log)

	req := &request.CoordinatedRequest{
		Method: "GET",
		URL:    "/hello/world",
		ID:     uuid.New().String(),
		Body:   []byte("world"),
		State: map[string][]byte{
			"file": []byte("main.md"),
		},
	}

	state, err := seq.exec(req)
	if err != nil {
		t.Error(err)
	}

	if val, ok := state.state["helloworld-rs"]; !ok {
		t.Error("helloworld state is missing")
	} else if !bytes.Equal(val, []byte("hello world")) {
		t.Error("unexpected helloworld state value:", string(val))
	}

	if val, ok := state.state["main.md"]; !ok {
		t.Error("get-file state is missing")
	} else if !bytes.Equal(val, []byte("## hello")) {
		t.Error("unexpected get-file state value:", string(val))
	}
}

func TestAsOnErrContinueSequence(t *testing.T) {
	steps := []directive.Executable{
		{
			CallableFn: directive.CallableFn{
				Fn: "helloworld-rs",
				As: "hello",
			},
		},
		{
			CallableFn: directive.CallableFn{
				Fn: "return-err",
				OnErr: &directive.FnOnErr{
					Any: "continue",
				},
			},
		},
	}

	seq := newSequence(steps, coord.grav.Connect, coord.log)

	req := &request.CoordinatedRequest{
		Method: "GET",
		URL:    "/hello/world",
		ID:     uuid.New().String(),
		Body:   []byte("world"),
		State:  map[string][]byte{},
	}

	state, err := seq.exec(req)
	if err != nil {
		t.Error(err)
	}

	if val, ok := state.state["hello"]; !ok {
		t.Error("hello state is missing")
	} else if !bytes.Equal(val, []byte("hello world")) {
		t.Error("unexpected hello state value:", string(val))
	}
}

func TestAsOnErrReturnSequence(t *testing.T) {
	steps := []directive.Executable{
		{
			CallableFn: directive.CallableFn{
				Fn: "helloworld-rs",
				As: "hello",
			},
		},
		{
			CallableFn: directive.CallableFn{
				Fn: "return-err",
				OnErr: &directive.FnOnErr{
					Any: "return",
				},
			},
		},
	}

	seq := newSequence(steps, coord.grav.Connect, coord.log)

	req := &request.CoordinatedRequest{
		Method: "GET",
		URL:    "/hello/world",
		ID:     uuid.New().String(),
		Body:   []byte("world"),
		State:  map[string][]byte{},
	}

	state, err := seq.exec(req)
	if err != ErrSequenceRunErr {
		t.Error(errors.New("sequence should have returned ErrSequenceRunErr, did not"))
	}

	if state.err.Code != 400 {
		t.Error("error code should be 400, is actually", state.err.Code)
	}

	if state.err.Message != "job failed" {
		t.Error("message should be 'job failed', is actually", state.err.Message)
	}
}

func TestWithSequence(t *testing.T) {
	steps := []directive.Executable{
		{
			CallableFn: directive.CallableFn{
				Fn: "helloworld-rs", // the body is empty, so this will return only "hello"
			},
		},
		{
			CallableFn: directive.CallableFn{
				Fn:   "modify-url", // if there's no body, it'll look in state for '
				With: map[string]string{"url": "helloworld-rs"},
			},
		},
	}

	seq := newSequence(steps, coord.grav.Connect, coord.log)

	req := &request.CoordinatedRequest{
		Method: "GET",
		URL:    "/hello/world",
		ID:     uuid.New().String(),
		Body:   []byte(""),
		State:  map[string][]byte{},
	}

	state, err := seq.exec(req)
	if err != nil {
		t.Error(err)
	}

	if val, ok := state.state["helloworld-rs"]; !ok {
		t.Error("helloworld-rs state is missing")
	} else if !bytes.Equal(val, []byte("hello ")) {
		t.Error("unexpected helloworld-rs state value:", string(val))
	}

	if val, ok := state.state["modify-url"]; !ok {
		t.Error("modify-url state is missing")
	} else if !bytes.Equal(val, []byte("hello /suborbital")) {
		t.Error("unexpected modify-url state value:", string(val))
	}
}

func TestForEachSequence(t *testing.T) {
	steps := []directive.Executable{
		{
			ForEach: &directive.ForEach{
				In: "people",
				Fn: "run-each",
				As: "hello-people",
			},
		},
	}

	seq := newSequence(steps, coord.grav.Connect, coord.log)

	req := &request.CoordinatedRequest{
		Method: "GET",
		URL:    "/hello/world",
		ID:     uuid.New().String(),
		Body:   []byte(""),
		State: map[string][]byte{
			"people": []byte(`[
				{
					"name": "Connor"
				},
				{
					"name": "Jimmy"
				},
				{
					"name": "Bob"
				}
			]`),
		},
	}

	state, err := seq.exec(req)
	if err != nil {
		t.Error(err)
	}

	val, ok := state.state["hello-people"]
	if !ok {
		t.Error("hello-people state is missing")
		return
	}

	stringVal := string(val)

	if !strings.Contains(stringVal, "{\"name\":\"Hello Jimmy\"}") {
		t.Error("unexpected hello-people state value:", string(val))
	}
}
