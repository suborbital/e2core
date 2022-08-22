package coordinator

import (
	"bytes"
	"testing"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/suborbital/appspec/appsource/bundle"
	"github.com/suborbital/appspec/request"
	"github.com/suborbital/appspec/tenant/executable"
	"github.com/suborbital/deltav/options"
	"github.com/suborbital/deltav/scheduler"
	"github.com/suborbital/deltav/server/coordinator/sequence"
	"github.com/suborbital/vektor/vk"
	"github.com/suborbital/vektor/vlog"
)

var coord *Coordinator

func init() {
	opts := options.NewWithModifiers(
		options.UseLogger(vlog.Default(
			vlog.Level(vlog.LogLevelDebug),
		)),
	)

	appSource := bundle.NewBundleSource("../../example-project/runnables.wasm.zip")

	coord = New(appSource, opts)

	if err := coord.Start(); err != nil {
		opts.Logger().Error(errors.Wrap(err, "failed to coord.Start"))
	}
}

func TestBasicSequence(t *testing.T) {
	steps := []executable.Executable{
		{
			ExecutableMod: executable.ExecutableMod{
				FQMN: "com.suborbital.test#default::helloworld-rs@v0.0.1",
			},
		},
	}

	req := &request.CoordinatedRequest{
		Method: "GET",
		URL:    "/hello/world",
		ID:     uuid.New().String(),
		Body:   []byte("world"),
		State:  map[string][]byte{},
	}

	seq, err := sequence.New(steps, req, vk.NewCtx(coord.log, nil, nil))
	if err != nil {
		t.Error(errors.Wrap(err, "failed to sequence.New"))
		return
	}

	if err := seq.Execute(coord.exec); err != nil {
		t.Error(err)
		return
	}

	if val, ok := req.State["helloworld-rs"]; !ok {
		t.Error("helloworld state is missing")
	} else if !bytes.Equal(val, []byte("hello world")) {
		t.Error("unexpected helloworld state value:", string(val))
	}
}

func TestGroupSequence(t *testing.T) {
	steps := []executable.Executable{
		{
			Group: []executable.ExecutableMod{
				{
					FQMN: "com.suborbital.test#default::helloworld-rs@v0.0.1",
				},
				{
					FQMN: "com.suborbital.test#default::get-file@v0.0.1",
					As:   "main.md",
				},
			},
		},
	}

	req := &request.CoordinatedRequest{
		Method: "GET",
		URL:    "/hello/world",
		ID:     uuid.New().String(),
		Body:   []byte("world"),
		State: map[string][]byte{
			"file": []byte("main.md"),
		},
	}

	seq, err := sequence.New(steps, req, vk.NewCtx(coord.log, nil, nil))
	if err != nil {
		t.Error(errors.Wrap(err, "failed to sequence.New"))
		return
	}

	if err := seq.Execute(coord.exec); err != nil {
		t.Error(err)
	}

	if val, ok := req.State["helloworld-rs"]; !ok {
		t.Error("helloworld state is missing")
	} else if !bytes.Equal(val, []byte("hello world")) {
		t.Error("unexpected helloworld state value:", string(val))
	}

	if val, ok := req.State["main.md"]; !ok {
		t.Error("get-file state is missing")
	} else if !bytes.Equal(val, []byte("## hello")) {
		t.Error("unexpected get-file state value:", string(val))
	}
}

func TestAsOnErrContinueSequence(t *testing.T) {
	steps := []executable.Executable{
		{
			ExecutableMod: executable.ExecutableMod{
				FQMN: "com.suborbital.test#default::helloworld-rs@v0.0.1",
				As:   "hello",
			},
		},
		{
			ExecutableMod: executable.ExecutableMod{
				FQMN: "com.suborbital.test#default::return-err@v0.0.1",
				OnErr: &executable.ErrHandler{
					Any: "continue",
				},
			},
		},
	}

	req := &request.CoordinatedRequest{
		Method: "GET",
		URL:    "/hello/world",
		ID:     uuid.New().String(),
		Body:   []byte("world"),
		State:  map[string][]byte{},
	}

	seq, err := sequence.New(steps, req, vk.NewCtx(coord.log, nil, nil))
	if err != nil {
		t.Error(errors.Wrap(err, "failed to sequence.New"))
		return
	}

	if err := seq.Execute(coord.exec); err != nil {
		t.Error(err)
	}

	if val, ok := req.State["hello"]; !ok {
		t.Error("hello state is missing")
	} else if !bytes.Equal(val, []byte("hello world")) {
		t.Error("unexpected hello state value:", string(val))
	}
}

func TestAsOnErrReturnSequence(t *testing.T) {
	steps := []executable.Executable{
		{
			ExecutableMod: executable.ExecutableMod{
				FQMN: "com.suborbital.test#default::helloworld-rs@v0.0.1",
				As:   "hello",
			},
		},
		{
			ExecutableMod: executable.ExecutableMod{
				FQMN: "com.suborbital.test#default::return-err@v0.0.1",
				OnErr: &executable.ErrHandler{
					Any: "return",
				},
			},
		},
	}

	req := &request.CoordinatedRequest{
		Method: "GET",
		URL:    "/hello/world",
		ID:     uuid.New().String(),
		Body:   []byte("world"),
		State:  map[string][]byte{},
	}

	seq, err := sequence.New(steps, req, vk.NewCtx(coord.log, nil, nil))
	if err != nil {
		t.Error(errors.Wrap(err, "failed to sequence.New"))
		return
	}

	if err = seq.Execute(coord.exec); err == nil {
		t.Error(errors.New("sequence should have returned err, did not"))
		return
	}

	runErr, isRunErr := err.(scheduler.RunErr)
	if !isRunErr {
		t.Error(errors.Wrap(err, "sequence should have returned RunErr, did not"))
	}

	if runErr.Code != 400 {
		t.Error("error code should be 400, is actually", runErr.Code)
	}

	if runErr.Message != "job failed" {
		t.Error("message should be 'job failed', is actually", runErr.Message)
	}
}

func TestWithSequence(t *testing.T) {
	steps := []executable.Executable{
		{
			ExecutableMod: executable.ExecutableMod{
				FQMN: "com.suborbital.test#default::helloworld-rs@v0.0.1",
			},
		},
		{
			ExecutableMod: executable.ExecutableMod{
				FQMN: "com.suborbital.test#default::modify-url@v0.0.1",
				With: map[string]string{"url": "helloworld-rs"},
			},
		},
	}

	req := &request.CoordinatedRequest{
		Method: "GET",
		URL:    "/hello/world",
		ID:     uuid.New().String(),
		Body:   []byte(""),
		State:  map[string][]byte{},
	}

	seq, err := sequence.New(steps, req, vk.NewCtx(coord.log, nil, nil))
	if err != nil {
		t.Error(errors.Wrap(err, "failed to sequence.New"))
		return
	}

	if err := seq.Execute(coord.exec); err != nil {
		t.Error(err)
	}

	if val, ok := req.State["helloworld-rs"]; !ok {
		t.Error("helloworld-rs state is missing")
	} else if !bytes.Equal(val, []byte("hello ")) {
		t.Error("unexpected helloworld-rs state value:", string(val))
	}

	if val, ok := req.State["modify-url"]; !ok {
		t.Error("modify-url state is missing")
	} else if !bytes.Equal(val, []byte("hello /suborbital")) {
		t.Error("unexpected modify-url state value:", string(val))
	}
}

func TestAsSequence(t *testing.T) {
	steps := []executable.Executable{
		{
			ExecutableMod: executable.ExecutableMod{
				FQMN: "com.suborbital.test#default::helloworld-rs@v0.0.1",
				As:   "url",
			},
		},
		{
			ExecutableMod: executable.ExecutableMod{
				FQMN: "com.suborbital.test#default::modify-url@v0.0.1",
			},
		},
	}

	req := &request.CoordinatedRequest{
		Method: "GET",
		URL:    "/hello/world",
		ID:     uuid.New().String(),
		Body:   []byte("friend"),
		State:  map[string][]byte{},
	}

	seq, err := sequence.New(steps, req, vk.NewCtx(coord.log, nil, nil))
	if err != nil {
		t.Error(errors.Wrap(err, "failed to sequence.New"))
		return
	}

	if err := seq.Execute(coord.exec); err != nil {
		t.Error(err)
	}

	if val, ok := req.State["url"]; !ok {
		t.Error("url state is missing")
	} else if !bytes.Equal(val, []byte("hello friend")) {
		t.Error("unexpected url state value:", string(val))
	}

	if val, ok := req.State["modify-url"]; !ok {
		t.Error("modify-url state is missing")
	} else if !bytes.Equal(val, []byte("hello friend/suborbital")) {
		t.Error("unexpected modify-url state value:", string(val))
	}
}
