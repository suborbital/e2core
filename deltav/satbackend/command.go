package satbackend

import (
	"crypto/rand"
	"fmt"
	"log"
	"math/big"

	"github.com/pkg/errors"

	"github.com/suborbital/appspec/tenant"
	"github.com/suborbital/deltav/options"
)

// satCommand returns the command and the port string
func satCommand(opts options.Options, module tenant.Module) (string, string) {
	port, err := randPort()
	if err != nil {
		log.Fatal(errors.Wrap(err, "failed to randPort"))
	}

	cmd := fmt.Sprintf(
		"sat %s",
		module.FQMN,
	)

	return cmd, port
}

func randPort() (string, error) {
	// choose a random port above 1000
	randPort, err := rand.Int(rand.Reader, big.NewInt(10000))
	if err != nil {
		return "", errors.Wrap(err, "failed to rand.Int")
	}

	return fmt.Sprintf("%d", randPort.Int64()+10000), nil
}
