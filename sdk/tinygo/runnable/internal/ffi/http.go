//go:build tinygo.wasm

package ffi

// #include <reactr.h>
import "C"

func DoHTTPRequest(method int32, urlStr string, body []byte, headers map[string]string) ([]byte, error) {
	urlPtr, urlSize := rawSlicePointer([]byte(urlStr))
	bodyPtr, bodySize := rawSlicePointer(body)

	size := C.fetch_url(method, urlPtr, urlSize, bodyPtr, bodySize, Ident())

	return result(size)
}
