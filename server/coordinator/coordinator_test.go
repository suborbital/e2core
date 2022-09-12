package coordinator

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/suborbital/appspec/appsource/bundle"
	"github.com/suborbital/appspec/request"
	"github.com/suborbital/appspec/tenant/executable"
	"github.com/suborbital/deltav/options"
	"github.com/suborbital/deltav/scheduler"
	"github.com/suborbital/deltav/server/coordinator/executor/mock"
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

	appSource := bundle.NewBundleSource("../../example-project/modules.wasm.zip")

	coord = New(appSource, opts)

	coord.exec = &mock.Executor{
		Jobs: map[string]mock.JobFunc{
			"/name/default/helloworld-rs": func(job interface{}, ctx *vk.Ctx) (interface{}, error) {
				req := job.(*request.CoordinatedRequest)
				resp := &request.CoordinatedResponse{
					Output: []byte(fmt.Sprintf("hello %s", string(req.Body))),
				}

				return resp, nil
			},
			"/name/default/get-file": func(job interface{}, ctx *vk.Ctx) (interface{}, error) {
				resp := &request.CoordinatedResponse{
					Output: []byte("## hello"),
				}

				return resp, nil
			},
			"/name/default/return-err": func(job interface{}, ctx *vk.Ctx) (interface{}, error) {
				return nil, scheduler.RunErr{Code: 400, Message: "job failed"}
			},
			"/name/default/modify-url": func(job interface{}, ctx *vk.Ctx) (interface{}, error) {
				req := job.(*request.CoordinatedRequest)
				urlState := req.State["url"]

				resp := &request.CoordinatedResponse{
					Output: []byte(fmt.Sprintf("%s/suborbital", string(urlState))),
				}

				return resp, nil
			},
		},
	}

	if err := coord.Start(); err != nil {
		opts.Logger().Error(errors.Wrap(err, "failed to coord.Start"))
	}
}

func TestBasicSequence(t *testing.T) {
	steps := []executable.Executable{
		{
			ExecutableMod: executable.ExecutableMod{
				FQMN: "/name/default/helloworld-rs",
			},
		},
	}

	req := &request.CoordinatedRequest{
		Method: "GET",
		URL:    "/name/default/helloworld-rs",
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

	if val, ok := req.State["/name/default/helloworld-rs"]; !ok {
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
					FQMN: "/name/default/helloworld-rs",
				},
				{
					FQMN: "/name/default/get-file",
					As:   "main.md",
				},
			},
		},
	}

	req := &request.CoordinatedRequest{
		Method: "GET",
		URL:    "/workflows/com.suborbital.test/default/testgroup",
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

	if val, ok := req.State["/name/default/helloworld-rs"]; !ok {
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
				FQMN: "/name/default/helloworld-rs",
				As:   "hello",
			},
		},
		{
			ExecutableMod: executable.ExecutableMod{
				FQMN: "/name/default/return-err",
				OnErr: &executable.ErrHandler{
					Any: "continue",
				},
			},
		},
	}

	req := &request.CoordinatedRequest{
		Method: "GET",
		URL:    "/workflows/com.suborbital.test/default/testasonerrcontinue",
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
				FQMN: "/name/default/helloworld-rs",
				As:   "hello",
			},
		},
		{
			ExecutableMod: executable.ExecutableMod{
				FQMN: "/name/default/return-err",
				OnErr: &executable.ErrHandler{
					Any: "return",
				},
			},
		},
	}

	req := &request.CoordinatedRequest{
		Method: "GET",
		URL:    "/workflows/com.suborbital.test/default/testasonerrreturn",
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
				FQMN: "/name/default/helloworld-rs",
			},
		},
		{
			ExecutableMod: executable.ExecutableMod{
				FQMN: "/name/default/modify-url",
				With: map[string]string{"url": "/name/default/helloworld-rs"},
			},
		},
	}

	req := &request.CoordinatedRequest{
		Method: "GET",
		URL:    "/workflows/com.suborbital.test/default/testwithsequence",
		ID:     uuid.New().String(),
		Body:   []byte("from URL"),
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

	if val, ok := req.State["/name/default/helloworld-rs"]; !ok {
		t.Error("helloworld-rs state is missing")
	} else if !bytes.Equal(val, []byte("hello from URL")) {
		t.Error("unexpected helloworld-rs state value:", string(val))
	}

	if val, ok := req.State["/name/default/modify-url"]; !ok {
		t.Error("modify-url state is missing")
	} else if !bytes.Equal(val, []byte("hello from URL/suborbital")) {
		t.Error("unexpected modify-url state value:", string(val))
	}
}

func TestAsSequence(t *testing.T) {
	steps := []executable.Executable{
		{
			ExecutableMod: executable.ExecutableMod{
				FQMN: "/name/default/helloworld-rs",
				As:   "url",
			},
		},
		{
			ExecutableMod: executable.ExecutableMod{
				FQMN: "/name/default/modify-url",
			},
		},
	}

	req := &request.CoordinatedRequest{
		Method: "GET",
		URL:    "/workflows/com.suborbital.test/default/testassequence",
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

	if val, ok := req.State["/name/default/modify-url"]; !ok {
		t.Error("modify-url state is missing")
	} else if !bytes.Equal(val, []byte("hello friend/suborbital")) {
		t.Error("unexpected modify-url state value:", string(val))
	}
}
