package api

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/pkg/errors"

	"github.com/suborbital/e2core/sat/engine2/runtime/instance"
)

const (
	methodGet int32 = iota
	methodHead
	methodOptions
	methodPost
	methodPut
	methodPatch
	methodDelete
)

const (
	contentTypeOctetStream = "application/octet-stream"
)

var methodValToMethod = map[int32]string{
	methodGet:     http.MethodGet,
	methodHead:    http.MethodHead,
	methodOptions: http.MethodOptions,
	methodPost:    http.MethodPost,
	methodPut:     http.MethodPut,
	methodPatch:   http.MethodPatch,
	methodDelete:  http.MethodDelete,
}

func (d *defaultAPI) FetchURLHandler() HostFn {
	fn := func(args ...interface{}) (interface{}, error) {
		method := args[0].(int32)
		urlPointer := args[1].(int32)
		urlSize := args[2].(int32)
		bodyPointer := args[3].(int32)
		bodySize := args[4].(int32)
		ident := args[5].(int32)

		ret := d.fetchUrl(method, urlPointer, urlSize, bodyPointer, bodySize, ident)

		return ret, nil
	}

	return NewHostFn("fetch_url", 6, true, fn)
}

func (d *defaultAPI) fetchUrl(method int32, urlPointer int32, urlSize int32, bodyPointer int32, bodySize int32, identifier int32) int32 {
	ll := d.logger.With().Str("method", "fetchUrl").Logger()

	// fetch makes a network request on bahalf of the wasm runner.
	// fetch writes the http response body into memory starting at returnBodyPointer, and the return value is a pointer to that memory
	inst, err := instance.ForIdentifier(identifier, true)
	if err != nil {
		ll.Err(err).Msg("instance.ForIdentifier")
		return -1
	}

	httpMethod, exists := methodValToMethod[method]
	if !exists {
		d.logger.Error().Int32("providedMethod", method).Msg("invalid method provided")
		return -2
	}

	urlBytes := inst.ReadMemory(urlPointer, urlSize)

	// the URL is encoded with headers added on the end, each seperated by ::
	// eg. https://google.com/somepage::authorization:bearer qdouwrnvgoquwnrg::anotherheader:nicetomeetyou
	urlParts := strings.Split(string(urlBytes), "::")
	urlString := urlParts[0]

	headers, err := parseHTTPHeaders(urlParts)
	if err != nil {
		ll.Err(err).Msg("parseHTTPHeaders")
		return -2
	}

	body := inst.ReadMemory(bodyPointer, bodySize)

	if len(body) > 0 {
		if headers.Get("Content-Type") == "" {
			headers.Add("Content-Type", contentTypeOctetStream)
		}
	}

	// wrap everything in a function so any errors get collected
	resp, err := func() ([]byte, error) {
		// filter the request through the capabilities
		resp, err := d.capabilities.HTTPClient.Do(d.capabilities.Auth, httpMethod, urlString, body, *headers)
		if err != nil {
			ll.Err(err).Msg("capabilities.HTTPClient.Do")
			return nil, err
		}

		defer resp.Body.Close()
		respBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			ll.Err(err).Msg("io.ReadAll response body")
		}

		if resp.StatusCode > 299 {
			ll.Debug().Int("status", resp.StatusCode).Msg("http request returned a non 1xx or 2xx status code")
			return nil, fmt.Errorf("%d: %s", resp.StatusCode, string(respBytes))
		}

		return respBytes, nil
	}()

	result, err := inst.Ctx().SetFFIResult(resp, err)
	if err != nil {
		ll.Err(err).Msg("inst.Ctx().SetFFIResult")
		return -1
	}

	return result.FFISize()
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
