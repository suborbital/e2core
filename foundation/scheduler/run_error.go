package scheduler

import (
	"encoding/json"
)

// RunErr represents an error returned from a Wasm Runnable
// it lives in the rt package to avoid import cycles
type RunErr struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

// Error returns the stringified JSON representation of the error
func (r RunErr) Error() string {
	bytes, _ := json.Marshal(r)
	return string(bytes)
}
