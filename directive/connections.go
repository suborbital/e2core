package directive

import (
	"fmt"
	"net/url"

	"github.com/pkg/errors"
	"github.com/suborbital/reactr/rcap"
)

const (
	dbTypeMySQL    = "mysql"
	dbTypePostgres = "postgresql"
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

type DBConnection struct {
	Type             string `yaml:"type" json:"type"`
	ConnectionString string `yaml:"connectionString" json:"connectionString"`
}

func (d *DBConnection) ToRCAPConfig(queries []DBQuery) (*rcap.DatabaseConfig, error) {
	if d == nil {
		return nil, nil
	}

	rcapType := rcap.DBTypeMySQL
	if d.Type == "postgresql" {
		rcapType = rcap.DBTypePostgres
	}

	rcapQueries := make([]rcap.Query, len(queries))
	for i := range queries {
		q, err := queries[i].toRCAPQuery(rcapType)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to toRCAPQuery for %s", queries[i].Name)
		}

		rcapQueries[i] = *q
	}

	config := &rcap.DatabaseConfig{
		Enabled:          d.ConnectionString != "",
		DBType:           rcapType,
		ConnectionString: d.ConnectionString,
		Queries:          rcapQueries,
	}

	return config, nil
}

func (d *DBConnection) validate() error {
	if d.Type != dbTypeMySQL && d.Type != dbTypePostgres {
		return fmt.Errorf("database type %s is invalid, must be 'mysql' or 'postgresql'", d.Type)
	}

	if d.ConnectionString == "" {
		return errors.New("database connectionString is empty")
	}

	return nil
}
