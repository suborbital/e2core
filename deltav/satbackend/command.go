package satbackend

import (
	"crypto/rand"
	"fmt"
	"log"
	"math/big"

	"github.com/pkg/errors"

	"github.com/suborbital/appspec/tenant"
	"github.com/suborbital/deltav/deltav/satbackend/config"
)

// satCommand returns the command and the port string
func satCommand(config config.Config, module tenant.Module) (string, string) {
	port, err := randPort()
	if err != nil {
		log.Fatal(errors.Wrap(err, "failed to randPort"))
	}

	var cmd string

	switch config.ExecMode {
	case "docker":
		cmd = fmt.Sprintf(
			"docker run --rm -p %s:%s -e SAT_HTTP_PORT=%s -e SAT_CONTROL_PLANE=docker.for.mac.localhost:9090 --network bridge suborbital/sat:%s sat %s",
			port, port, port,
			config.SatTag,
			module.FQMN,
		)
	case "metal":
		cmd = fmt.Sprintf(
			"sat %s",
			module.FQMN,
		)
	default:
		cmd = "echo 'invalid exec mode'"
	}

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
