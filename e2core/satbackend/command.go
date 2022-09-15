package satbackend

import (
	"crypto/rand"
	"fmt"
	"log"
	"math/big"

	"github.com/pkg/errors"

	"github.com/suborbital/appspec/tenant"
)

// modStartCommand returns the command and the port string
func modStartCommand(module tenant.Module) (string, string) {
	port, err := randPort()
	if err != nil {
		log.Fatal(errors.Wrap(err, "failed to randPort"))
	}

	cmd := fmt.Sprintf("e2core mod start %s", module.FQMN)

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
