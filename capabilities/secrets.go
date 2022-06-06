package capabilities

// SecretsCapability controls access to secrets
type SecretsCapability interface {
	GetSecretValue(key string) string
}

// SecretsConfig is the configuration for environment secrets
type SecretsConfig struct {
	Enabled bool              `json:"enabled" yaml:"enabled"`
	Env     *EnvSecretsConfig `json:"env,omitempty" yaml:"enabled,omitempty"`
}

type EnvSecretsConfig struct {
	AllowedKeys []string `json:"allowedKeys" yaml:"allowedKeys"`
}
