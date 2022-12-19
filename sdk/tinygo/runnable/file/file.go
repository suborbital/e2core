//go:build tinygo.wasm

package file

import (
	"github.com/suborbital/reactr/api/tinygo/runnable/internal/ffi"
)

// Bytes fetches a []byte of a requested static file. Writing to this slice
// does not modify its contents.
func Bytes(filename string) ([]byte, error) {
	return ffi.GetStaticFile(filename)
}
