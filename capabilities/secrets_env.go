package capabilities

import (
	"os"

	"golang.org/x/exp/slices"
)

type envSecretsSource struct {
	config SecretsConfig
}

// NewEnvSecretsSource returns a environment-based secrets source
func NewEnvSecretsSource(config SecretsConfig) SecretsCapability {
	e := &envSecretsSource{
		config: config,
	}

	return e
}

// GetSecretValue returns the secret value for the given key
func (e *envSecretsSource) GetSecretValue(key string) string {
	if !e.config.Enabled {
		return ""
	}

	if !slices.Contains(e.config.Env.AllowedKeys, key) {
		return ""
	}

	val := os.Getenv(key)

	return val
}
