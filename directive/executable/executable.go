package executable

import (
	"errors"
)

var (
	// ErrSequenceShouldReturn is represents a failed function call that should result in a return.
	ErrSequenceShouldReturn = errors.New("function resulted in a Run Error and sequence should return")
	ErrSequenceCompleted    = errors.New("sequence is complete, no steps to run")
)

// Executable represents an executable step in a handler
// The 'ForEach' type has been disabled and removed as of Atmo v0.4.0.
type Executable struct {
	CallableFn `yaml:"callableFn,inline" json:"callableFn"`
	Group      []CallableFn `yaml:"group,omitempty" json:"group,omitempty"`
	ForEach    interface{}  `yaml:"forEach,omitempty"`
}

// CallableFn is a fn along with its "variable name" and "args".
type CallableFn struct {
	Fn    string            `yaml:"fn,omitempty" json:"fn,omitempty"`
	As    string            `yaml:"as,omitempty" json:"as,omitempty"`
	With  map[string]string `yaml:"with,omitempty" json:"with,omitempty"`
	OnErr *ErrHandler       `yaml:"onErr,omitempty" json:"onErr,omitempty"`
	FQFN  string            `yaml:"-" json:"fqfn"` // calculated during Validate.
}

// ErrHandler describes how to handle an error from a function call.
type ErrHandler struct {
	Code  map[int]string `yaml:"code,omitempty" json:"code,omitempty"`
	Any   string         `yaml:"any,omitempty" json:"any,omitempty"`
	Other string         `yaml:"other,omitempty" json:"other,omitempty"`
}

// IsGroup returns true if the executable is a group.
func (e Executable) IsGroup() bool {
	return e.Fn == "" && e.Group != nil && len(e.Group) > 0
}

// IsFn returns true if the executable is a group.
func (e Executable) IsFn() bool {
	return e.Fn != "" && e.Group == nil
}

func (c CallableFn) Key() string {
	key := c.Fn

	if c.As != "" {
		key = c.As
	}

	return key
}

func (c CallableFn) ShouldReturn(code int) error {
	// if the developer hasn't specified an error handler,
	// the default is to return.
	if c.OnErr == nil {
		return ErrSequenceShouldReturn
	}

	shouldErr := true

	// if the error code is listed as return, or any/other indicates a return, then create an erroring state object and return it.

	if len(c.OnErr.Code) > 0 {
		if val, ok := c.OnErr.Code[code]; ok && val == "continue" {
			shouldErr = false
		} else if !ok && c.OnErr.Other == "continue" {
			shouldErr = false
		}
	} else if c.OnErr.Any == "continue" {
		shouldErr = false
	}

	if shouldErr {
		return ErrSequenceShouldReturn
	}

	return nil
}
