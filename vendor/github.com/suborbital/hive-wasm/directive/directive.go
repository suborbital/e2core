package directive

import (
	"errors"
	"fmt"

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
	Identifier string     `yaml:"identifier"`
	Version    string     `yaml:"version"`
	Functions  []Function `yaml:"functions"`
	Handlers   []Handler  `yaml:"handlers,omitempty"`

	// "fully qualified function names"
	fqfns map[string]string `yaml:"-"`
}

// Function describes a function present inside of a bundle
type Function struct {
	Name      string
	NameSpace string
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
	Group []string `yaml:"group,omitempty"`
	Fn    string   `yaml:"fn,omitempty"`
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

	if !semver.IsValid(d.Version) {
		problems.add(errors.New("version is not a valid semver"))
	}

	if len(d.Functions) < 1 {
		problems.add(errors.New("no functions listed"))
	}

	fns := map[string]bool{}

	for i, f := range d.Functions {
		namespaced := fmt.Sprintf("%s#%s", f.NameSpace, f.Name)

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
		if f.NameSpace == "" {
			problems.add(fmt.Errorf("function at position %d missing namespace", i))
		}

		// if the fn is in the default namespace, let it exist "naked" and namespaced
		if f.NameSpace == NamespaceDefault {
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

		for j, s := range h.Steps {
			if !s.IsFn() && !s.IsGroup() {
				problems.add(fmt.Errorf("step at position %d for handler handler at position %d has neither Fn or Group", j, i))
			}

			if s.IsFn() {
				if _, exists := fns[s.Fn]; !exists {
					problems.add(fmt.Errorf("handler at positiion %d lists fn at step %d that does not exist: %s (did you forget a namespace?)", i, j, s.Fn))
				}
			} else if s.IsGroup() {
				for k, gfn := range s.Group {
					if _, exists := fns[gfn]; !exists {
						problems.add(fmt.Errorf("handler at positiion %d lists fn at position %d in group at step %d that does not exist: %s (did you forget a namespace?)", i, k, j, gfn))
					}
				}
			}
		}

		lastStep := h.Steps[len(h.Steps)-1]
		if h.Response == "" && lastStep.IsGroup() {
			problems.add(fmt.Errorf("handler at position %d has group as last step but does not include 'response' field", i))
		} else if h.Response != "" {
			if _, exists := fns[h.Response]; !exists {
				problems.add(fmt.Errorf("handler at positiion %d lists response fn name that does not exist: %s", i, h.Response))
			}
		}
	}

	return problems.render()
}

func (d *Directive) calculateFQFNs() {
	d.fqfns = map[string]string{}

	for _, fn := range d.Functions {
		namespaced := fmt.Sprintf("%s#%s", fn.NameSpace, fn.Name)

		// if the function is in the default namespace, add it to the map both namespaced and not
		if fn.NameSpace == NamespaceDefault {
			d.fqfns[fn.Name] = d.fqfnForFunc(fn.NameSpace, fn.Name)
			d.fqfns[namespaced] = d.fqfnForFunc(fn.NameSpace, fn.Name)
		} else {
			d.fqfns[namespaced] = d.fqfnForFunc(fn.NameSpace, fn.Name)
		}
	}
}

func (d *Directive) fqfnForFunc(namespace, fn string) string {
	return fmt.Sprintf("%s#%s@%s", namespace, fn, d.Version)
}

// IsGroup returns true if the executable is a group
func (e *Executable) IsGroup() bool {
	return e.Fn == "" && e.Group != nil && len(e.Group) > 0
}

// IsFn returns true if the executable is a group
func (e *Executable) IsFn() bool {
	return e.Fn != "" && e.Group == nil
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
