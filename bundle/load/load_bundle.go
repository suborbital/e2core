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

	if err := Runnables(r, bundle.Directive.Runnables); err != nil {
		return errors.Wrap(err, "failed to ModuleRefsIntoInstance")
	}

	return nil
}

// Runnables loads a set of WasmModuleRefs into a Reactr instance
// if you're trying to use this directly, you probably want BundleFromPath or Bundle instead
func Runnables(r *rt.Reactr, runnables []directive.Runnable) error {
	for i, runnable := range runnables {
		if runnable.ModuleRef == nil {
			return fmt.Errorf("missing ModuleRef for Runnable %s", runnable.Name)
		}

		// this func is an odd but needed optimization;
		// if neither of the two `r.Register` calls
		// below end up getting called, we don't want
		// to create the Runner, since that adds things
		// to Reactr's global state, which would be a waste.
		getRunner := func() *rwasm.Runner {
			return rwasm.NewRunnerWithRef(runnables[i].ModuleRef)
		}

		// if the Runnable's 'simple name' is already registered, de-register it
		// as we don't want to 'overwrite', since that will leave resources stranded
		if r.IsRegistered(runnable.Name) {
			// this can error, but for now we can't really
			// fail if this does, it would break several things
			r.DeRegister(runnable.Name)
		}

		r.Register(runnable.Name, getRunner())

		// load the Runnable under its FQFN as well because
		// that's what will be called when a sequence runs
		if runnable.FQFN != "" {
			// if a module is already registered, don't bother over-writing
			// since FQFNs are 'guaranteed' to be unique, so there's no point
			if !r.IsRegistered(runnable.FQFN) {
				r.Register(runnable.FQFN, getRunner(), rt.PreWarm(), rt.Autoscale(0))
			}
		}
	}

	return nil
}
