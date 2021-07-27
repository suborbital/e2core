package rcap

import (
	"os"
	"strings"
)

// AuthProvider is a provider for various kinds of auth
type AuthProvider interface {
	HeaderForDomain(string) *AuthHeader
}

// AuthHeader is an HTTP header designed to authenticate requests
type AuthHeader struct {
	HeaderType string `json:"headerType"`
	Value      string `json:"value"`
}

// AuthProviderConfig is a config for the default auth provider
type AuthProviderConfig struct {
	// Headers is a map between domains and auth header that should be added to requests to those domains
	Headers map[string]AuthHeader `json:"headers"`
}

type defaultAuthProvider struct {
	config *AuthProviderConfig

	augmentedHeaders map[string]AuthHeader
}

// DefaultAuthProvider creates the default static auth provider
func DefaultAuthProvider(config *AuthProviderConfig) AuthProvider {
	ap := &defaultAuthProvider{
		config:           config,
		augmentedHeaders: map[string]AuthHeader{},
	}

	return ap
}

// HeadersForDomain returns the appropriate auth headers for the given domain
func (ap *defaultAuthProvider) HeaderForDomain(domain string) *AuthHeader {
	header, ok := ap.augmentedHeaders[domain]
	if !ok {
		if ap.config == nil {
			return nil
		}

		origignalHeader, exists := ap.config.Headers[domain]
		if !exists {
			return nil
		}

		augmented := augmentHeaderFromEnv(origignalHeader)

		ap.augmentedHeaders[domain] = augmented
		header = augmented
	}

	return &header
}

// augmentHeadersFromEnv takes a an AuthHeader and replaces and `env()` values with their representative values from the environment
func augmentHeaderFromEnv(header AuthHeader) AuthHeader {
	// turn env(SOME_HEADER_KEY) into SOME_HEADER_KEY
	if strings.HasPrefix(header.Value, "env(") && strings.HasSuffix(header.Value, ")") {
		headerKey := strings.TrimPrefix(header.Value, "env(")
		headerKey = strings.TrimSuffix(headerKey, ")")

		val := os.Getenv(headerKey)

		augmentedHeader := AuthHeader{
			HeaderType: header.HeaderType,
			Value:      val,
		}

		return augmentedHeader
	}

	return header
}
