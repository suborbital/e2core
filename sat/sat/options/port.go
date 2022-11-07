package options

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/pkg/errors"
	"github.com/sethvargo/go-envconfig"
)

var _ envconfig.Decoder = (*port)(nil) // interface check

type port string

// EnvDecode implements the envconfig.Decoder interface for the port.
func (p *port) EnvDecode(in string) error {
	if in != "" {
		*p = port(in)
		return nil
	}

	// choose a random port above 1000
	randPort, err := rand.Int(rand.Reader, big.NewInt(10000))
	if err != nil {
		return errors.Wrap(err, "failed to rand.Int")
	}

	*p = port(fmt.Sprintf("%d", randPort.Int64()+1000))

	return nil
}
