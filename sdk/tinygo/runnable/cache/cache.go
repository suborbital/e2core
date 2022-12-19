//go:build tinygo.wasm

package cache

import (
	"github.com/suborbital/reactr/api/tinygo/runnable/internal/ffi"
)

func Get(key string) ([]byte, error) {
	return ffi.CacheGet(key)
}

func Set(key, val string, ttl int) {
	ffi.CacheSet(key, val, ttl)
}
