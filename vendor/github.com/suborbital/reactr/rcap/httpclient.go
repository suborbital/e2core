package rcap

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
)

// HTTPClient gives Runnables the ability to make HTTP requests
type HTTPClient interface {
	Do(auth AuthProvider, method, urlString string, body []byte, headers http.Header) (*http.Response, error)
}

type defaultHTTPClient struct{}

// DefaultHTTPClient creates an HTTP client with no restrictions
func DefaultHTTPClient() HTTPClient {
	d := &defaultHTTPClient{}

	return d
}

// Do performs the provided request
func (d *defaultHTTPClient) Do(auth AuthProvider, method, urlString string, body []byte, headers http.Header) (*http.Response, error) {
	urlObj, err := url.Parse(urlString)
	if err != nil {
		return nil, errors.Wrap(err, "failed to url.Parse")
	}

	req, err := http.NewRequest(method, urlObj.String(), bytes.NewBuffer(body))
	if err != nil {
		return nil, errors.Wrap(err, "failed to NewRequest")
	}

	authHeader := auth.HeaderForDomain(urlObj.Host)
	if authHeader != nil && authHeader.Value != "" {
		headers.Add("Authorization", fmt.Sprintf("%s %s", authHeader.HeaderType, authHeader.Value))
	}

	req.Header = headers

	return http.DefaultClient.Do(req)
}
