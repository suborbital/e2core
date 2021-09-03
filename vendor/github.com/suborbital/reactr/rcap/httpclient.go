package rcap

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
)

// HTTPConfig is configuration for the HTTP capability
type HTTPConfig struct {
	Enabled bool      `json:"enabled" yaml:"enabled"`
	Rules   HTTPRules `json:"rules" yaml:"rules"`
}

// HTTPCapability gives Runnables the ability to make HTTP requests
type HTTPCapability interface {
	Do(auth AuthCapability, method, urlString string, body []byte, headers http.Header) (*http.Response, error)
}

type httpClient struct {
	config HTTPConfig
}

// DefaultHTTPClient creates an HTTP client with no restrictions
func DefaultHTTPClient(config HTTPConfig) HTTPCapability {
	d := &httpClient{
		config: config,
	}

	return d
}

// Do performs the provided request
func (h *httpClient) Do(auth AuthCapability, method, urlString string, body []byte, headers http.Header) (*http.Response, error) {
	if !h.config.Enabled {
		return nil, ErrCapabilityNotEnabled
	}

	urlObj, err := url.Parse(urlString)
	if err != nil {
		return nil, errors.Wrap(err, "failed to url.Parse")
	}

	req, err := http.NewRequest(method, urlObj.String(), bytes.NewBuffer(body))
	if err != nil {
		return nil, errors.Wrap(err, "failed to NewRequest")
	}

	if err := h.config.Rules.requestIsAllowed(req); err != nil {
		return nil, errors.Wrap(err, "failed to requestIsAllowed")
	}

	authHeader := auth.HeaderForDomain(urlObj.Host)
	if authHeader != nil && authHeader.Value != "" {
		headers.Add("Authorization", fmt.Sprintf("%s %s", authHeader.HeaderType, authHeader.Value))
	}

	req.Header = headers

	return http.DefaultClient.Do(req)
}
