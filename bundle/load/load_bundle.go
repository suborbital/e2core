package load

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/suborbital/atmo/bundle"
	"github.com/suborbital/atmo/directive"
	"github.com/suborbital/reactr/rt"
	"github.com/suborbital/reactr/rwasm"
)

// BundleFromPath loads a .wasm.zip file into the rt instance
func BundleFromPath(r *rt.Reactr, path string) error {
	if !strings.HasSuffix(path, ".wasm.zip") {
		return fmt.Errorf("cannot load bundle %s, does not have .wasm.zip extension", filepath.Base(path))
	}

	bundle, err := bundle.Read(path)
	if err != nil {
		return errors.Wrap(err, "failed to ReadBundle")
	}

	if err := Bundle(r, bundle); err != nil {
		return errors.Wrap(err, "failed to IntoInstance")
	}

	return nil
}

// Bundle loads a .wasm.zip file into the rt instance
func Bundle(r *rt.Reactr, bundle *bundle.Bundle) error {
	if err := bundle.Directive.Validate(); err != nil {
		return errors.Wrap(err, "failed to Validate bundle directive")
	}

	if err := Runnables(r, bundle.Directive.Runnables, bundle.StaticFile); err != nil {
		return errors.Wrap(err, "failed to ModuleRefsIntoInstance")
	}

	return nil
}

// Runnables loads a set of WasmModuleRefs into a Reactr instance
func Runnables(r *rt.Reactr, runnables []directive.Runnable, staticFileFunc rwasm.FileFunc) error {
	for i, runnable := range runnables {
		if runnable.ModuleRef == nil {
			return fmt.Errorf("missing ModuleRef for Runnable %s", runnable.Name)
		}

		runner := rwasm.NewRunnerWithRef(runnables[i].ModuleRef, staticFileFunc)

		// pre-warm so that Runnables have at least one instance active
		// when the first request is received.
		r.Register(runnable.Name, runner, rt.PreWarm())

		// mount the fqfn if possible in case multiple
		// bundles with conflicting names get mounted.
		if runnable.FQFN != "" {
			r.Register(runnable.FQFN, runner, rt.PreWarm())
		}
	}

	return nil
}
