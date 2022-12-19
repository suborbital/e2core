//go:build tinygo.wasm

package ffi

import "errors"

type HostErr error

func NewHostError(message string) HostErr {
	return errors.New(message).(HostErr)
}
