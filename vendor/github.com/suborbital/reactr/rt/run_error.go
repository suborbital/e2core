package rt

import (
	"encoding/json"

	"github.com/suborbital/vektor/vk"
)

// RunErr represents an error returned from a Wasm Runnable
// it lives in the rt package to avoid import cycles
type RunErr struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Error returns the stringified JSON representation of the error
func (r RunErr) Error() string {
	bytes, _ := json.Marshal(r)
	return string(bytes)
}

// ToVKErr converts a RunErr to a VKError
func (r RunErr) ToVKErr() vk.Error {
	return vk.Err(r.Code, r.Message)
}
