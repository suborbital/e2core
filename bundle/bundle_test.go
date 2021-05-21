package bundle

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pkg/errors"
)

func TestRead(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Error(errors.Wrap(err, "failed to get CWD"))
	}

	bundle, err := Read(filepath.Join(cwd, "../example-project/runnables.wasm.zip"))
	if err != nil {
		t.Error(errors.Wrap(err, "failed to Read"))
		return
	}

	if len(bundle.ModuleRefs) == 0 {
		t.Error("bundle had 0 runnables")
		return
	}

	hasDefault := false
	for _, r := range bundle.ModuleRefs {
		if r.Name == "helloworld-rs.wasm" {
			hasDefault = true
		}
	}

	if !hasDefault {
		t.Error("helloworld-rs.wasm runnable not found in bundle")
	}
}
