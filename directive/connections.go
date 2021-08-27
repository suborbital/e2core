package directive

import (
	"net/url"

	"github.com/pkg/errors"
	"github.com/suborbital/reactr/rcap"
)

// NATSConnection describes a connection to a NATS server
type NATSConnection struct {
	ServerAddress string `yaml:"serverAddress" json:"serverAddress"`
}

func (n *NATSConnection) validate() error {
	if n.ServerAddress == "" {
		return errors.New("serverAddress is empty")
	}

	if _, err := url.Parse(n.ServerAddress); err != nil {
		return errors.Wrap(err, "failed to parse serverAddress as URL")
	}

	return nil
}

func validateRedisConfig(config *rcap.RedisConfig) error {
	if config.ServerAddress == "" {
		return errors.New("serverAddress is empty")
	}

	if _, err := url.Parse(config.ServerAddress); err != nil {
		return errors.Wrap(err, "failed to parse serverAddress as URL")
	}

	return nil
}
