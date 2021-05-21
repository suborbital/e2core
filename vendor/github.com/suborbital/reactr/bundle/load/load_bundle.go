package load

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/suborbital/reactr/bundle"
	"github.com/suborbital/reactr/rt"
	"github.com/suborbital/reactr/rwasm"
	"github.com/suborbital/reactr/rwasm/moduleref"
)

// IntoInstanceFromPath loads a .wasm.zip file into the rt instance
func IntoInstanceFromPath(r *rt.Reactr, path string) error {
	if !strings.HasSuffix(path, ".wasm.zip") {
		return fmt.Errorf("cannot load bundle %s, does not have .wasm.zip extension", filepath.Base(path))
	}

	bundle, err := bundle.Read(path)
	if err != nil {
		return errors.Wrap(err, "failed to ReadBundle")
	}

	if err := IntoInstance(r, bundle); err != nil {
		return errors.Wrap(err, "failed to IntoInstance")
	}

	return nil
}

// IntoInstance loads a .wasm.zip file into the rt instance
func IntoInstance(r *rt.Reactr, bundle *bundle.Bundle) error {
	if err := bundle.Directive.Validate(); err != nil {
		return errors.Wrap(err, "failed to Validate bundle directive")
	}

	if err := ModuleRefsIntoInstance(r, bundle.ModuleRefs, bundle.StaticFile); err != nil {
		return errors.Wrap(err, "failed to ModuleRefsIntoInstance")
	}

	return nil
}

// ModuleRefsIntoInstance loads a set of WasmModuleRefs into a Reactr instance
func ModuleRefsIntoInstance(r *rt.Reactr, refs []moduleref.WasmModuleRef, staticFileFunc rwasm.FileFunc) error {
	for i, ref := range refs {
		runner := rwasm.NewRunnerWithRef(&refs[i], staticFileFunc)

		jobName := strings.TrimSuffix(ref.Name, ".wasm")

		// pre-warm so that Runnables have at least one instance active
		// when the first request is received.
		r.Register(jobName, runner, rt.PreWarm())

		// mount the fqfn if possible in case multiple
		// bundles with conflicting names get mounted.
		if ref.FQFN != "" {
			r.Register(ref.FQFN, runner, rt.PreWarm())
		}

	}

	return nil
}
