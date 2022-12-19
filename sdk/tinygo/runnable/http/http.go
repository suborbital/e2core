//go:build tinygo.wasm

package http

import (
	"github.com/suborbital/reactr/api/tinygo/runnable/internal/ffi"

	"github.com/suborbital/reactr/api/tinygo/runnable/http/method"
)

func GET(url string, headers map[string]string) ([]byte, error) {
	return do(method.GET, url, nil, headers)
}

func HEAD(url string, headers map[string]string) ([]byte, error) {
	return do(method.HEAD, url, nil, headers)
}

func OPTIONS(url string, headers map[string]string) ([]byte, error) {
	return do(method.OPTIONS, url, nil, headers)
}

func POST(url string, body []byte, headers map[string]string) ([]byte, error) {
	return do(method.POST, url, body, headers)
}

func PUT(url string, body []byte, headers map[string]string) ([]byte, error) {
	return do(method.PUT, url, body, headers)
}

func PATCH(url string, body []byte, headers map[string]string) ([]byte, error) {
	return do(method.PATCH, url, body, headers)
}

func DELETE(url string, headers map[string]string) ([]byte, error) {
	return do(method.DELETE, url, nil, headers)
}

// Remark: The URL gets encoded with headers added on the end, seperated by ::
// eg. https://google.com/somepage::authorization:bearer qdouwrnvgoquwnrg::anotherheader:nicetomeetyou
func do(method method.MethodType, url string, body []byte, headers map[string]string) ([]byte, error) {
	urlStr := url

	if headers != nil {
		headerStr := renderHeaderString(headers)
		if headerStr != "" {
			urlStr += "::" + headerStr
		}
	}

	return ffi.DoHTTPRequest(int32(method), urlStr, body, headers)
}

func renderHeaderString(headers map[string]string) string {
	out := ""

	for key, value := range headers {
		out += key + ":" + value
		out += "::"
	}

	return out[:len(out)-2]
}
