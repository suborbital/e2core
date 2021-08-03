package rwasm

import (
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"github.com/wasmerio/wasmer-go/wasmer"
)

const (
	methodGet    = int32(1)
	methodPost   = int32(2)
	methodPatch  = int32(3)
	methodDelete = int32(4)
)

const (
	contentTypeJSON        = "application/json"
	contentTypeTextPlain   = "text/plain"
	contentTypeOctetStream = "application/octet-stream"
)

var methodValToMethod = map[int32]string{
	methodGet:    http.MethodGet,
	methodPost:   http.MethodPost,
	methodPatch:  http.MethodPatch,
	methodDelete: http.MethodDelete,
}

func fetchURL() *HostFn {
	fn := func(args ...wasmer.Value) (interface{}, error) {
		method := args[0].I32()
		urlPointer := args[1].I32()
		urlSize := args[2].I32()
		bodyPointer := args[3].I32()
		bodySize := args[4].I32()
		ident := args[5].I32()

		ret := fetch_url(method, urlPointer, urlSize, bodyPointer, bodySize, ident)

		return ret, nil
	}

	return newHostFn("fetch_url", 6, true, fn)
}

func fetch_url(method int32, urlPointer int32, urlSize int32, bodyPointer int32, bodySize int32, identifier int32) int32 {
	// fetch makes a network request on bahalf of the wasm runner.
	// fetch writes the http response body into memory starting at returnBodyPointer, and the return value is a pointer to that memory
	inst, err := instanceForIdentifier(identifier, true)
	if err != nil {
		internalLogger.Error(errors.Wrap(err, "[rwasm] alert: invalid identifier used, potential malicious activity"))
		return -1
	}

	httpMethod, exists := methodValToMethod[method]
	if !exists {
		internalLogger.ErrorString("invalid method provided: ", method)
		return -2
	}

	urlBytes := inst.readMemory(urlPointer, urlSize)

	// the URL is encoded with headers added on the end, each seperated by ::
	// eg. https://google.com/somepage::authorization:bearer qdouwrnvgoquwnrg::anotherheader:nicetomeetyou
	urlParts := strings.Split(string(urlBytes), "::")
	urlString := urlParts[0]

	headers, err := parseHTTPHeaders(urlParts)
	if err != nil {
		internalLogger.Error(errors.Wrap(err, "could not parse URL headers"))
		return -2
	}

	body := inst.readMemory(bodyPointer, bodySize)

	if len(body) > 0 {
		if headers.Get("Content-Type") == "" {
			headers.Add("Content-Type", contentTypeOctetStream)
		}
	}

	// filter the request through the capabilities
	resp, err := inst.ctx.HTTPClient.Do(inst.ctx.Auth, httpMethod, urlString, body, *headers)
	if err != nil {
		internalLogger.Error(errors.Wrap(err, "failed to Do request"))
		return -3
	}

	if resp.StatusCode > 299 {
		internalLogger.Debug("runnable's http request returned non-200 response:", resp.StatusCode)
		return int32(resp.StatusCode) * -1 // return a negative value, i.e. -404 for a 404 error
	}

	defer resp.Body.Close()
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		internalLogger.Error(errors.Wrap(err, "failed to Read response body"))
		return -4
	}

	inst.setFFIResult(respBytes)

	return int32(len(respBytes))
}

func parseHTTPHeaders(urlParts []string) (*http.Header, error) {
	headers := &http.Header{}

	if len(urlParts) > 1 {
		for _, p := range urlParts[1:] {
			headerParts := strings.Split(p, ":")
			if len(headerParts) != 2 {
				return nil, errors.New("header was not formatted correctly")
			}

			headers.Add(headerParts[0], headerParts[1])
		}
	}

	return headers, nil
}
