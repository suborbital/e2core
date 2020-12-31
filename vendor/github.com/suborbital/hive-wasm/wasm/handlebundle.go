package wasm

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/suborbital/hive-wasm/bundle"
	"github.com/suborbital/hive/hive"
)

// HandleBundleAtPath loads a .wasm.zip file into the hive instance
func HandleBundleAtPath(h *hive.Hive, path string) error {
	if !strings.HasSuffix(path, ".wasm.zip") {
		return fmt.Errorf("cannot load bundle %s, does not have .wasm.zip extension", filepath.Base(path))
	}

	bundle, err := bundle.Read(path)
	if err != nil {
		return errors.Wrap(err, "failed to ReadBundle")
	}

	return HandleBundle(h, bundle)
}

// HandleBundle loads a .wasm.zip file into the hive instance
func HandleBundle(h *hive.Hive, bundle *bundle.Bundle) error {
	if err := bundle.Directive.Validate(); err != nil {
		return errors.Wrap(err, "failed to Validate bundle directive")
	}

	for i, r := range bundle.Runnables {
		runner := newRunnerWithRef(&bundle.Runnables[i])

		jobName := strings.Replace(r.Name, ".wasm", "", -1)
		fqfn, err := bundle.Directive.FQFN(jobName)
		if err != nil {
			return errors.Wrapf(err, "failed to FQFN for %s", jobName)
		}

		// mount both the "raw" name and the fqfn in case
		// multiple bundles with conflicting names get mounted
		h.Handle(jobName, runner)
		h.Handle(fqfn, runner)

	}

	return nil
}
