package directive

import (
	"errors"
	"fmt"
	"strings"

	"golang.org/x/mod/semver"
	"gopkg.in/yaml.v2"
)

// InputTypeRequest and others represent consts for Directives
const (
	InputTypeRequest = "request"
)

// NamespaceDefault and others represent conts for namespaces
const (
	NamespaceDefault = "default"
)

// Directive describes a set of functions and a set of handlers
// that take an input, and compose a set of functions to handle it
type Directive struct {
	Identifier  string     `yaml:"identifier"`
	AppVersion  string     `yaml:"appVersion"`
	AtmoVersion string     `yaml:"atmoVersion"`
	Runnables   []Runnable `yaml:"runnables"`
	Handlers    []Handler  `yaml:"handlers,omitempty"`

	// "fully qualified function names"
	fqfns map[string]string `yaml:"-"`
}

// Handler represents the mapping between an input and a composition of functions
type Handler struct {
	Input    Input `yaml:"input,inline"`
	Steps    []Executable
	Response string `yaml:"response,omitempty"`
}

// Input represents an input source
type Input struct {
	Type     string
	Method   string
	Resource string
}

// Executable represents an executable step in a handler
type Executable struct {
	CallableFn `yaml:"callableFn,inline"`
	Group      []CallableFn `yaml:"group,omitempty"`
}

// CallableFn is a fn along with its "variable name" and "args"
type CallableFn struct {
	Fn           string   `yaml:"fn,omitempty"`
	As           string   `yaml:"as,omitempty"`
	With         []string `yaml:"with,omitempty"`
	DesiredState []Alias  `yaml:"-"`
}

// Alias is the parsed version of an entry in the `With` array from a CallableFn
// If you do user: activeUser, then activeUser is the state key and user
// is the key that gets put into the function's state (i.e. the alias)
type Alias struct {
	Key   string
	Alias string
}

// Marshal outputs the YAML bytes of the Directive
func (d *Directive) Marshal() ([]byte, error) {
	return yaml.Marshal(d)
}

// Unmarshal unmarshals YAML bytes into a Directive struct
// it also calculates a map of FQFNs for later use
func (d *Directive) Unmarshal(in []byte) error {
	return yaml.Unmarshal(in, d)
}

// FQFN returns the FQFN for a given function in the directive
func (d *Directive) FQFN(fn string) (string, error) {
	if d.fqfns == nil {
		d.calculateFQFNs()
	}

	fqfn, exists := d.fqfns[fn]
	if !exists {
		return "", fmt.Errorf("fn %s does not exist", fn)
	}

	return fqfn, nil
}

// Validate validates a directive
func (d *Directive) Validate() error {
	problems := &problems{}

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
		namespaced := fmt.Sprintf("%s#%s", f.Namespace, f.Name)

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
		if f.Namespace == NamespaceDefault {
			fns[f.Name] = true
			fns[namespaced] = true
		} else {
			fns[namespaced] = true
		}
	}

	for i, h := range d.Handlers {
		if h.Input.Type == "" {
			problems.add(fmt.Errorf("handler at position %d missing type", i))
		}

		if h.Input.Resource == "" {
			problems.add(fmt.Errorf("handler at position %d missing resource", i))
		}

		if h.Input.Type == InputTypeRequest && h.Input.Method == "" {
			problems.add(fmt.Errorf("handler at position %d is of type request, but does not specify a method", i))
		}

		if len(h.Steps) == 0 {
			problems.add(fmt.Errorf("handler at position %d missing steps", i))
			continue
		}

		// keep track of the functions that have run so far at each step
		fullState := map[string]bool{}

		for j, s := range h.Steps {
			fnsToAdd := []string{}

			if !s.IsFn() && !s.IsGroup() {
				problems.add(fmt.Errorf("step at position %d for handler handler at position %d has neither Fn or Group", j, i))
			}

			validateFn := func(fn CallableFn) {
				if _, exists := fns[fn.Fn]; !exists {
					problems.add(fmt.Errorf("handler at positiion %d lists fn at step %d that does not exist: %s (did you forget a namespace?)", i, j, s.Fn))
				}

				if _, err := fn.ParseWith(); err != nil {
					problems.add(fmt.Errorf("handler at position %d has invalid 'with' value at step %d: %s", i, j, err.Error()))
				}

				for _, d := range fn.DesiredState {
					if _, exists := fullState[d.Key]; !exists {
						problems.add(fmt.Errorf("handler at position %d has 'with' value at step %d referencing a key that is not yet available in the handler's state: %s", i, j, d.Key))
					}
				}

				key := fn.Fn
				if fn.As != "" {
					key = fn.As
				}

				fnsToAdd = append(fnsToAdd, key)
			}

			if s.IsFn() {
				validateFn(s.CallableFn)
			} else {
				for _, gfn := range s.Group {
					validateFn(gfn)
				}
			}

			for _, newFn := range fnsToAdd {
				fullState[newFn] = true
			}
		}

		lastStep := h.Steps[len(h.Steps)-1]
		if h.Response == "" && lastStep.IsGroup() {
			problems.add(fmt.Errorf("handler at position %d has group as last step but does not include 'response' field", i))
		} else if h.Response != "" {
			if _, exists := fullState[h.Response]; !exists {
				problems.add(fmt.Errorf("handler at positiion %d lists response state key that does not exist: %s", i, h.Response))
			}
		}
	}

	return problems.render()
}

func (d *Directive) calculateFQFNs() {
	d.fqfns = map[string]string{}

	for _, fn := range d.Runnables {
		namespaced := fmt.Sprintf("%s#%s", fn.Namespace, fn.Name)

		// if the function is in the default namespace, add it to the map both namespaced and not
		if fn.Namespace == NamespaceDefault {
			d.fqfns[fn.Name] = d.fqfnForFunc(fn.Namespace, fn.Name)
			d.fqfns[namespaced] = d.fqfnForFunc(fn.Namespace, fn.Name)
		} else {
			d.fqfns[namespaced] = d.fqfnForFunc(fn.Namespace, fn.Name)
		}
	}
}

func (d *Directive) fqfnForFunc(namespace, fn string) string {
	return fmt.Sprintf("%s#%s@%s", namespace, fn, d.AppVersion)
}

// IsGroup returns true if the executable is a group
func (e *Executable) IsGroup() bool {
	return e.Fn == "" && e.Group != nil && len(e.Group) > 0
}

// IsFn returns true if the executable is a group
func (e *Executable) IsFn() bool {
	return e.Fn != "" && e.Group == nil
}

// ParseWith parses the fn's 'with' clause and returns the desired state
func (c *CallableFn) ParseWith() ([]Alias, error) {
	if c.DesiredState != nil && len(c.DesiredState) > 0 {
		return c.DesiredState, nil
	}

	c.DesiredState = make([]Alias, len(c.With))

	for i, w := range c.With {
		parts := strings.Split(w, ": ")
		if len(parts) != 2 {
			return nil, fmt.Errorf("with value has wrong format: parsed %d parts seperated by : , expected 2", len(parts))
		}

		c.DesiredState[i] = Alias{Alias: parts[0], Key: parts[1]}
	}

	return c.DesiredState, nil
}

type problems []error

func (p *problems) add(err error) {
	*p = append(*p, err)
}

func (p *problems) render() error {
	if len(*p) == 0 {
		return nil
	}

	text := fmt.Sprintf("found %d problems:", len(*p))

	for _, err := range *p {
		text += fmt.Sprintf("\n\t%s", err.Error())
	}

	return errors.New(text)
}
