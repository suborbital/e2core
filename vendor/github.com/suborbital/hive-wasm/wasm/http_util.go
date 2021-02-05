package wasm

import (
	"net/http"
	"strings"

	"github.com/pkg/errors"
)

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
