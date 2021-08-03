package load

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/suborbital/atmo/bundle"
	"github.com/suborbital/atmo/directive"
	"github.com/suborbital/reactr/rcap"
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

	var authConfig *rcap.AuthProviderConfig
	if bundle.Directive.Authentication != nil && bundle.Directive.Authentication.Domains != nil {
		// need to convert the Directive headers to rcap headers
		// rcap should add yaml tags so we don't need this
		headers := map[string]rcap.AuthHeader{}
		for k, v := range bundle.Directive.Authentication.Domains {
			headers[k] = rcap.AuthHeader{
				HeaderType: v.HeaderType,
				Value:      v.Value,
			}
		}

		authConfig = &rcap.AuthProviderConfig{Headers: headers}
	}

	if err := Runnables(r, bundle.Directive.Runnables, authConfig, bundle.StaticFile, true); err != nil {
		return errors.Wrap(err, "failed to ModuleRefsIntoInstance")
	}

	return nil
}

// Runnables loads a set of WasmModuleRefs into a Reactr instance
// if you're trying to use this directly, you probably want BundleFromPath or Bundle instead
func Runnables(r *rt.Reactr, runnables []directive.Runnable, authConfig *rcap.AuthProviderConfig, staticFileFunc rcap.StaticFileFunc, registerSimpleName bool) error {
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

		// take the default capabilites from the Reactr instance
		caps := r.DefaultCaps()
		// set our own FileSource that is connected to the Bundle's FileFunc
		caps.FileSource = rcap.DefaultFileSource(staticFileFunc)
		// set our own auth provider based on the Directive
		if authConfig != nil {
			caps.Auth = rcap.DefaultAuthProvider(authConfig)
		}

		// TODO: in the future, this should be updated to
		// de-register a Runnable if one with the same name
		// is already registered, since over-registering can
		// cause workers to languish in the background
		if registerSimpleName {
			r.RegisterWithCaps(runnable.Name, getRunner(), caps)
		}

		// we load the Runnable under its FQFN because
		// that's what will be called when a sequence runs
		if runnable.FQFN != "" {
			// if a module is already registered, don't bother over-writing
			// since FQFNs are 'guaranteed' to be unique, so there's no point
			if !r.IsRegistered(runnable.FQFN) {
				r.RegisterWithCaps(runnable.FQFN, getRunner(), caps, rt.PreWarm())
			}
		}
	}

	return nil
}
