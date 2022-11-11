package options

import (
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sethvargo/go-envconfig"
)

var _ envconfig.Decoder = (*procUUID)(nil) // interface check

type procUUID string

// EnvDecode implements the envconfig.Decoder interface for the port.
func (p *procUUID) EnvDecode(in string) error {
	if in == "" {
		*p = procUUID(uuid.New().String())
		return nil
	}

	parsedUUID, err := uuid.Parse(in)
	if err != nil {
		return errors.Wrap(err, "SAT_UUID is set, but is not valid UUID")
	}

	*p = procUUID(parsedUUID.String())

	return nil
}
