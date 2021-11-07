package directive

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/suborbital/atmo/fqfn"
	"golang.org/x/mod/semver"
)

// Validate validates a directive
func (d *Directive) Validate() error {
	problems := &problems{}

	d.calculateFQFNs()

	if d.Identifier == "" {
		problems.add(errors.New("identifier is missing"))
	}

	if !semver.IsValid(d.AppVersion) {
		problems.add(errors.New("app version is not a valid semantic version"))
	}

	if !semver.IsValid(d.AtmoVersion) {
		problems.add(errors.New("atmo version is not a valid semantic version"))
	}

	if len(d.Runnables) < 1 {
		problems.add(errors.New("no functions listed"))
	}

	fns := map[string]bool{}

	for i, f := range d.Runnables {
		namespaced := fmt.Sprintf("%s::%s", f.Namespace, f.Name)

		if _, exists := fns[namespaced]; exists {
			problems.add(fmt.Errorf("duplicate fn %s found", namespaced))
			continue
		}

		if _, exists := fns[f.Name]; exists {
			problems.add(fmt.Errorf("duplicate fn %s found", namespaced))
			continue
		}

		if f.Name == "" {
			problems.add(fmt.Errorf("function at position %d missing name", i))
			continue
		}

		if f.Namespace == "" {
			problems.add(fmt.Errorf("function at position %d missing namespace", i))
		}

		// if the fn is in the default namespace, let it exist "naked" and namespaced
		if f.Namespace == fqfn.NamespaceDefault {
			fns[f.Name] = true
			fns[namespaced] = true
		} else {
			fns[namespaced] = true
		}
	}

	// validate connections before handlers because we want to make sure they're all correct first
	if d.Connections != nil {
		if d.Connections.NATS != nil {
			if err := d.Connections.NATS.validate(); err != nil {
				problems.add(err)
			}
		}

		if d.Connections.Redis != nil {
			if err := validateRedisConfig(d.Connections.Redis); err != nil {
				problems.add(err)
			}
		}

		if d.Connections.Database != nil {
			if err := d.Connections.Database.validate(); err != nil {
				problems.add(err)
			}
		}
	}

	if d.Authentication != nil {
		if d.Authentication.Domains != nil {
			for d, h := range d.Authentication.Domains {
				if h.HeaderType == "" {
					h.HeaderType = "bearer"
				}

				if h.Value == "" {
					problems.add(fmt.Errorf("authentication for domain %s has an empty value", d))
				}
			}
		}
	}

	for i, h := range d.Handlers {
		if h.Input.Type != InputTypeRequest && h.Input.Type != InputTypeStream {
			problems.add(fmt.Errorf("handler for resource %s has invalid type, must be 'request' or 'stream'", h.Input.Resource))
		}

		if h.Input.Resource == "" {
			problems.add(fmt.Errorf("handler at position %d missing resource", i))
		}

		if h.Input.Type == InputTypeRequest {
			if !strings.HasPrefix(h.Input.Resource, "/") {
				problems.add(fmt.Errorf("handler resource must begin with leading slash '/': %s", h.Input.Resource))
			}

			if h.Input.Method == "" {
				problems.add(fmt.Errorf("handler for resource %s has type 'request', but does not specify a method", h.Input.Resource))
			}

			if h.RespondTo != "" {
				problems.add(fmt.Errorf("handler for resource %s has type 'request', but defines a 'respondTo' field, which only valid for type 'stream'", h.Input.Resource))
			}
		} else if h.Input.Type == InputTypeStream {
			if h.Input.Source == "" || h.Input.Source == InputSourceServer {
				if !strings.HasPrefix(h.Input.Resource, "/") {
					problems.add(fmt.Errorf("handler resource must begin with leading slash '/': %s", h.Input.Resource))
				}
			} else if h.Input.Source == InputSourceNATS {
				if d.Connections == nil || d.Connections.NATS == nil {
					problems.add(fmt.Errorf("handler for resource %s references source %s that is not configured", h.Input.Resource, h.Input.Source))
				}
			} else {
				problems.add(fmt.Errorf("handler for resource %s references source %s that does not exist", h.Input.Resource, h.Input.Source))
			}
		}

		if len(h.Steps) == 0 {
			problems.add(fmt.Errorf("handler for resource %s missing steps", h.Input.Resource))
			continue
		}

		name := fmt.Sprintf("%s %s", h.Input.Method, h.Input.Resource)
		fullState := d.validateSteps(executableTypeHandler, name, h.Steps, map[string]bool{}, problems)

		lastStep := h.Steps[len(h.Steps)-1]
		if h.Response == "" && lastStep.IsGroup() {
			problems.add(fmt.Errorf("handler for %s has group as last step but does not include 'response' field", name))
		} else if h.Response != "" {
			if _, exists := fullState[h.Response]; !exists {
				problems.add(fmt.Errorf("handler for %s lists response state key that does not exist: %s", name, h.Response))
			}
		}
	}

	for i, s := range d.Schedules {
		if s.Name == "" {
			problems.add(fmt.Errorf("schedule at position %d has no name", i))
			continue
		}

		if len(s.Steps) == 0 {
			problems.add(fmt.Errorf("schedule %s missing steps", s.Name))
			continue
		}

		if s.Every.Seconds == 0 && s.Every.Minutes == 0 && s.Every.Hours == 0 && s.Every.Days == 0 {
			problems.add(fmt.Errorf("schedule %s has no 'every' values", s.Name))
		}

		// user can provide an 'initial state' via the schedule.State field, so let's prime the state with it.
		initialState := map[string]bool{}
		for k := range s.State {
			initialState[k] = true
		}

		d.validateSteps(executableTypeSchedule, s.Name, s.Steps, initialState, problems)
	}

	for i, q := range d.Queries {
		if q.Name == "" {
			problems.add(fmt.Errorf("query at position %d has no name", i))
		}

		if q.Query == "" {
			problems.add(fmt.Errorf("query at position %d has no query value", i))
		}

		if q.Type != "" {
			if q.Type != queryTypeInsert && q.Type != queryTypeSelect {
				problems.add(fmt.Errorf("query at position %d has invalid type %s", i, q.Type))
			}
		}

		if q.VarCount < 0 {
			problems.add(fmt.Errorf("query at position %d cannot have negative var count", i))
		}
	}

	return problems.render()
}

type executableType string

const (
	executableTypeHandler  = executableType("handler")
	executableTypeSchedule = executableType("schedule")
)

func (d *Directive) validateSteps(exType executableType, name string, steps []Executable, initialState map[string]bool, problems *problems) map[string]bool {
	// keep track of the functions that have run so far at each step
	fullState := initialState

	for j, s := range steps {
		fnsToAdd := []string{}

		if !s.IsFn() && !s.IsGroup() {
			if s.ForEach != nil {
				problems.add(fmt.Errorf("step at position %d for %s %s is a 'forEach', which was removed in v0.4.0", j, exType, name))
			} else {
				problems.add(fmt.Errorf("step at position %d for %s %s isn't an Fn or Group", j, exType, name))
			}
		}

		// this function is key as it compartmentalizes 'step validation', and importantly it
		// ensures that a Runnable is available to handle it and binds it by setting the FQFN field
		validateFn := func(fn *CallableFn) {
			runnable := d.FindRunnable(fn.Fn)
			if runnable == nil {
				problems.add(fmt.Errorf("%s for %s lists fn at step %d that does not exist: %s (did you forget a namespace?)", exType, name, j, fn.Fn))
			} else {
				fn.FQFN = runnable.FQFN
			}

			for _, key := range fn.With {
				if _, exists := fullState[key]; !exists {
					problems.add(fmt.Errorf("%s for %s has 'with' value at step %d referencing a key that is not yet available in the handler's state: %s", exType, name, j, key))
				}
			}

			if fn.OnErr != nil {
				// if codes are specificed, 'other' should be used, not 'any'
				if len(fn.OnErr.Code) > 0 && fn.OnErr.Any != "" {
					problems.add(fmt.Errorf("%s for %s has 'onErr.any' value at step %d while specific codes are specified, use 'other' instead", exType, name, j))
				} else if fn.OnErr.Any != "" {
					if fn.OnErr.Any != "continue" && fn.OnErr.Any != "return" {
						problems.add(fmt.Errorf("%s for %s has 'onErr.any' value at step %d with an invalid error directive: %s", exType, name, j, fn.OnErr.Any))
					}
				}

				// if codes are NOT specificed, 'any' should be used, not 'other'
				if len(fn.OnErr.Code) == 0 && fn.OnErr.Other != "" {
					problems.add(fmt.Errorf("%s for %s has 'onErr.other' value at step %d while specific codes are not specified, use 'any' instead", exType, name, j))
				} else if fn.OnErr.Other != "" {
					if fn.OnErr.Other != "continue" && fn.OnErr.Other != "return" {
						problems.add(fmt.Errorf("%s for %s has 'onErr.any' value at step %d with an invalid error directive: %s", exType, name, j, fn.OnErr.Other))
					}
				}

				for code, val := range fn.OnErr.Code {
					if val != "return" && val != "continue" {
						problems.add(fmt.Errorf("%s for %s has 'onErr.code' value at step %d with an invalid error directive for code %d: %s", exType, name, j, code, val))
					}
				}
			}

			key := fn.Fn
			if fn.As != "" {
				key = fn.As
			}

			fnsToAdd = append(fnsToAdd, key)
		}

		// the steps below are referenced by index (j) to ensure the addition of the FQFN in validateFn 'sticks'
		if s.IsFn() {
			validateFn(&steps[j].CallableFn)
		} else if s.IsGroup() {
			for p := range s.Group {
				validateFn(&steps[j].Group[p])
			}
		}

		for _, newFn := range fnsToAdd {
			fullState[newFn] = true
		}
	}

	return fullState
}
