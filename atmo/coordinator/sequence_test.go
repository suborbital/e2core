package coordinator

import (
	"bytes"
	"log"
	"testing"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/suborbital/reactr/bundle"
	"github.com/suborbital/reactr/directive"
	"github.com/suborbital/reactr/request"
	"github.com/suborbital/vektor/vlog"
)

var coord *Coordinator

func init() {
	coord = New(vlog.Default())

	bundle, err := bundle.Read("../../example-project/runnables.wasm.zip")
	if err != nil {
		log.Fatal(errors.Wrap(err, "failed to Read bundle"))
	}

	coord.UseBundle(bundle)
}

func TestBasicSequence(t *testing.T) {
	steps := []directive.Executable{
		{
			CallableFn: directive.CallableFn{
				Fn: "helloworld-rs",
			},
		},
	}

	if coord.bundle == nil {
		t.Error("directive is nil")
		return
	}

	seq := newSequence(steps, coord.grav.Connect, coord.bundle.Directive.FQFN, coord.log)

	req := &request.CoordinatedRequest{
		Method: "GET",
		URL:    "/hello/world",
		ID:     uuid.New().String(),
		Body:   []byte("world"),
		State: map[string][]byte{
			"hello": []byte("what is up"),
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
}
