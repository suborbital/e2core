package rcap

import "net/http"

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type defaultHTTPClient struct{}

// DefaultHTTPClient creates an HTTP client with no restrictions
func DefaultHTTPClient() HTTPClient {
	d := &defaultHTTPClient{}

	return d
}

// Do performs the provided request
func (d *defaultHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return http.DefaultClient.Do(req)
}
