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

// Directive describes a set of functions and a set of handlers
// that take an input, and compose a set of functions to handle it
type Directive struct {
	Identifier string     `yaml:"identifier"`
	Version    string     `yaml:"version"`
	Functions  []Function `yaml:"functions"`
	Handlers   []Handler  `yaml:"handlers,omitempty"`
}

// Marshal outputs the YAML bytes of the Directive
func (d *Directive) Marshal() ([]byte, error) {
	return yaml.Marshal(d)
}

// Unmarshal unmarshals YAML bytes into a Directive struct
func (d *Directive) Unmarshal(in []byte) error {
	return yaml.Unmarshal(in, d)
}

// Function describes a function present inside of a bundle
type Function struct {
	Name      string
	NameSpace string
}

// Handler represents the mapping between an input and a composition of functions
type Handler struct {
	Input Input `yaml:"input,inline"`
	Steps []Executable
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

	for i, f := range d.Functions {
		if f.Name == "" {
			problems.add(fmt.Errorf("function at position %d missing name", i))
		}
		if f.NameSpace == "" {
			problems.add(fmt.Errorf("function at position %d missing namespace", i))
		}
	}

	if len(d.Handlers) > 0 {
		for i, h := range d.Handlers {
			if h.Input.Type == "" {
				problems.add(fmt.Errorf("handler at position %d missing type", i))
			}
			if h.Input.Resource == "" {
				problems.add(fmt.Errorf("handler at position %d missing resource", i))
			}

			if len(h.Steps) == 0 {
				problems.add(fmt.Errorf("handler at position %d missing steps", i))
				continue
			}

			for j, s := range h.Steps {
				if s.Fn == "" && (s.Group == nil || len(s.Group) == 0) {
					problems.add(fmt.Errorf("step at position %d for handler handler at position %d has neither Fn or Group", j, i))
				}
			}
		}
	}

	return problems.render()
}
