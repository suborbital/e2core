package directive

import (
	"errors"
	"fmt"
	"strings"

	"github.com/suborbital/atmo/fqfn"
	"github.com/suborbital/reactr/rcap"
	"golang.org/x/mod/semver"
	"gopkg.in/yaml.v2"
)

// InputTypeRequest and others represent consts for Directives
const (
	InputTypeRequest  = "request"
	InputTypeStream   = "stream"
	InputSourceServer = "server"
	InputSourceNATS   = "nats"
)

// Directive describes a set of functions and a set of handlers
// that take an input, and compose a set of functions to handle it
type Directive struct {
	Identifier     string                 `yaml:"identifier" json:"identifier"`
	AppVersion     string                 `yaml:"appVersion" json:"appVersion"`
	AtmoVersion    string                 `yaml:"atmoVersion" json:"atmoVersion"`
	Headless       bool                   `yaml:"headless,omitempty" json:"headless,omitempty"`
	Connections    *Connections           `yaml:"connections,omitempty" json:"connections,omitempty"`
	Authentication *Authentication        `yaml:"authentication,omitempty" json:"authentication,omitempty"`
	Capabilities   *rcap.CapabilityConfig `yaml:"capabilities,omitempty" json:"capabilities,omitempty"`
	Handlers       []Handler              `yaml:"handlers,omitempty" json:"handlers,omitempty"`
	Schedules      []Schedule             `yaml:"schedules,omitempty" json:"schedules,omitempty"`

	// Runnables is populated by subo, never by the user
	Runnables []Runnable `yaml:"runnables" json:"runnables"`
}

// Handler represents the mapping between an input and a composition of functions
type Handler struct {
	Input     Input        `yaml:"input,inline" json:"input"`
	Steps     []Executable `yaml:"steps" json:"steps"`
	Response  string       `yaml:"response,omitempty" json:"response,omitempty"`
	RespondTo string       `yaml:"respondTo,omitempty" json:"respondTo,omitempty"`
}

// Schedule represents the mapping between an input and a composition of functions
type Schedule struct {
	Name  string            `yaml:"name" json:"name"`
	Every ScheduleEvery     `yaml:"every" json:"every"`
	State map[string]string `yaml:"state,omitempty" json:"state,omitempty"`
	Steps []Executable      `yaml:"steps" json:"steps"`
}

// ScheduleEvery represents the 'every' value for a schedule
type ScheduleEvery struct {
	Seconds int `yaml:"seconds,omitempty" json:"seconds,omitempty"`
	Minutes int `yaml:"minutes,omitempty" json:"minutes,omitempty"`
	Hours   int `yaml:"hours,omitempty" json:"hours,omitempty"`
	Days    int `yaml:"days,omitempty" json:"days,omitempty"`
}

// Input represents an input source
type Input struct {
	Type     string `yaml:"type" json:"type"`
	Source   string `yaml:"source,omitempty" json:"source,omitempty"`
	Method   string `yaml:"method" json:"method"`
	Resource string `yaml:"resource" json:"resource"`
}

// Executable represents an executable step in a handler
type Executable struct {
	CallableFn `yaml:"callableFn,inline" json:"callableFn"`
	Group      []CallableFn `yaml:"group,omitempty" json:"group,omitempty"`
	ForEach    *ForEach     `yaml:"forEach,omitempty" json:"forEach,omitempty"`
}

// CallableFn is a fn along with its "variable name" and "args"
type CallableFn struct {
	Fn    string            `yaml:"fn,omitempty" json:"fn,omitempty"`
	As    string            `yaml:"as,omitempty" json:"as,omitempty"`
	With  map[string]string `yaml:"with,omitempty" json:"with,omitempty"`
	OnErr *FnOnErr          `yaml:"onErr,omitempty" json:"onErr,omitempty"`
	FQFN  string            `yaml:"-" json:"fqfn"` // calculated during Validate
}

// FnOnErr describes how to handle an error from a function call
type FnOnErr struct {
	Code  map[int]string `yaml:"code,omitempty" json:"code,omitempty"`
	Any   string         `yaml:"any,omitempty" json:"any,omitempty"`
	Other string         `yaml:"other,omitempty" json:"other,omitempty"`
}

// ForEach describes a forEach operator
type ForEach struct {
	In         string     `yaml:"in" json:"in"`
	Fn         string     `yaml:"fn" json:"fn"`
	As         string     `yaml:"as" json:"as"`
	OnErr      *FnOnErr   `yaml:"onErr,omitempty" json:"onErr,omitempty"`
	CallableFn CallableFn `yaml:"-" json:"callableFn"` // calculated during Validate
}

// Connections describes connections
type Connections struct {
	NATS  *NATSConnection   `yaml:"nats,omitempty" json:"nats,omitempty"`
	Redis *rcap.RedisConfig `yaml:"redis,omitempty" json:"redis,omitempty"`
}

type Authentication struct {
	Domains map[string]rcap.AuthHeader `yaml:"domains,omitempty" json:"domains,omitempty"`
}

func (d *Directive) FindRunnable(name string) *Runnable {
	// if this is an FQFN, parse the identifier and bail out
	// if it doesn't match this Directive

	FQFN := fqfn.Parse(name)

	if FQFN.Identifier != "" && FQFN.Identifier != d.Identifier {
		return nil
	}

	if FQFN.Version != "" && FQFN.Version != d.AppVersion {
		return nil
	}

	for i, r := range d.Runnables {
		if r.Name == FQFN.Fn && r.Namespace == FQFN.Namespace {
			return &d.Runnables[i]
		}
	}

	return nil
}

// Marshal outputs the YAML bytes of the Directive
func (d *Directive) Marshal() ([]byte, error) {
	d.calculateFQFNs()

	return yaml.Marshal(d)
}

// Unmarshal unmarshals YAML bytes into a Directive struct
// it also calculates a map of FQFNs for later use
func (d *Directive) Unmarshal(in []byte) error {
	if err := yaml.Unmarshal(in, d); err != nil {
		return err
	}

	d.calculateFQFNs()

	return nil
}

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

		if !s.IsFn() && !s.IsGroup() && !s.IsForEach() {
			problems.add(fmt.Errorf("step at position %d for %s %s isn't an Fn, Group, or ForEach", j, exType, name))
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
		} else if s.IsForEach() {
			if s.ForEach.In == "" {
				problems.add(fmt.Errorf("ForEach at position %d for %s %s is missing 'in' value", j, exType, name))
			}

			if s.ForEach.As == "" {
				problems.add(fmt.Errorf("ForEach at position %d for %s %s is missing 'as' value", j, exType, name))
			}

			steps[j].ForEach.CallableFn = CallableFn{Fn: s.ForEach.Fn, OnErr: s.ForEach.OnErr, As: s.ForEach.As}
			validateFn(&s.ForEach.CallableFn)
		}

		for _, newFn := range fnsToAdd {
			fullState[newFn] = true
		}
	}

	return fullState
}

func (d *Directive) calculateFQFNs() {
	for i, fn := range d.Runnables {
		if fn.FQFN != "" {
			continue
		}

		if fn.Namespace == "" {
			fn.Namespace = fqfn.NamespaceDefault
		}

		if fn.Version == "" {
			fn.Version = d.AppVersion
		}

		d.Runnables[i].FQFN = d.fqfnForFunc(fn.Namespace, fn.Name)
	}
}

func (d *Directive) fqfnForFunc(namespace, fn string) string {
	return fqfn.FromParts(d.Identifier, namespace, fn, d.AppVersion)
}

// NumberOfSeconds calculates the total time in seconds for the schedule's 'every' value
func (s *Schedule) NumberOfSeconds() int {
	seconds := s.Every.Seconds
	minutes := 60 * s.Every.Minutes
	hours := 60 * 60 * s.Every.Hours
	days := 60 * 60 * 24 * s.Every.Days

	return seconds + minutes + hours + days
}

// IsGroup returns true if the executable is a group
func (e *Executable) IsGroup() bool {
	return e.Fn == "" && e.Group != nil && len(e.Group) > 0 && e.ForEach == nil
}

// IsFn returns true if the executable is a group
func (e *Executable) IsFn() bool {
	return e.Fn != "" && e.Group == nil && e.ForEach == nil
}

// IsForEach returns true if the exectuable is a ForEach
func (e *Executable) IsForEach() bool {
	return e.ForEach != nil && e.Fn == "" && e.Group == nil
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
