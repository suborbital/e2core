package capabilities

// AuthCapability is a provider for various kinds of auth
type AuthCapability interface {
	HeaderForDomain(string) *AuthHeader
}

// AuthConfig is a config for the default auth provider
type AuthConfig struct {
	Enabled bool `json:"enabled" yaml:"enabled"`

	// Headers is a map between domains and auth header that should be added to requests to those domains
	Headers map[string]AuthHeader `json:"headers" yaml:"headers"`
}

// AuthHeader is an HTTP header designed to authenticate requests
type AuthHeader struct {
	HeaderType string `json:"headerType" yaml:"headerType"`
	Value      string `json:"value" yaml:"value"`
}

type defaultAuthProvider struct {
	config AuthConfig

	augmentedHeaders map[string]AuthHeader
}

// DefaultAuthProvider creates the default static auth provider
func DefaultAuthProvider(config AuthConfig) AuthCapability {
	ap := &defaultAuthProvider{
		config:           config,
		augmentedHeaders: map[string]AuthHeader{},
	}

	return ap
}

// HeadersForDomain returns the appropriate auth headers for the given domain
func (ap *defaultAuthProvider) HeaderForDomain(domain string) *AuthHeader {
	if !ap.config.Enabled {
		return nil
	}

	header, ok := ap.augmentedHeaders[domain]
	if !ok {
		if ap.config.Headers == nil {
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

// augmentHeadersFromEnv takes a an AuthHeader and replaces any
// `env()` values with their representative values from the environment
func augmentHeaderFromEnv(header AuthHeader) AuthHeader {
	augmentedHeader := AuthHeader{
		HeaderType: header.HeaderType,
		Value:      AugmentedValFromEnv(header.Value),
	}

	return augmentedHeader
}
