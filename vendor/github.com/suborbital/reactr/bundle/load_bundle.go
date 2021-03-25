package bundle

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/suborbital/reactr/rt"
	"github.com/suborbital/reactr/rwasm"
)

// LoadFromPath loads a .wasm.zip file into the rt instance
func LoadFromPath(h *rt.Reactr, path string) error {
	if !strings.HasSuffix(path, ".wasm.zip") {
		return fmt.Errorf("cannot load bundle %s, does not have .wasm.zip extension", filepath.Base(path))
	}

	bundle, err := Read(path)
	if err != nil {
		return errors.Wrap(err, "failed to ReadBundle")
	}

	return Load(h, bundle)
}

// Load loads a .wasm.zip file into the rt instance
func Load(h *rt.Reactr, bundle *Bundle) error {
	if err := bundle.Directive.Validate(); err != nil {
		return errors.Wrap(err, "failed to Validate bundle directive")
	}

	for i, r := range bundle.Runnables {
		runner := rwasm.NewRunnerWithRef(&bundle.Runnables[i], bundle.StaticFile)

		jobName := strings.TrimSuffix(r.Name, ".wasm")
		fqfn, err := bundle.Directive.FQFN(jobName)
		if err != nil {
			return errors.Wrapf(err, "failed to FQFN for %s", jobName)
		}

		// mount both the "raw" name and the fqfn in case
		// multiple bundles with conflicting names get mounted.

		// pre-warm so that Runnables have at least one instance active
		// when the first request is received.
		h.Handle(jobName, runner, rt.PreWarm())
		h.Handle(fqfn, runner, rt.PreWarm())

	}

	return nil
}
