package sat

import (
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/suborbital/e2core/bus/bus"
	"github.com/suborbital/vektor/vlog"
)

func connectStaticPeers(logger *vlog.Logger, b *bus.Bus) error {
	count := 0
	var err error

	endpoints, useStatic := os.LookupEnv("SAT_PEERS")
	if useStatic {
		epts := strings.Split(endpoints, ",")

		for _, e := range epts {
			logger.Debug("connecting to static peer", e)

			for count < 10 {
				if err = b.ConnectEndpoint(e); err != nil {
					logger.Error(errors.Wrap(err, "failed to ConnectEndpoint, will retry"))
					count++

					time.Sleep(time.Second * 3)
				} else {
					break
				}
			}
		}
	}

	return err
}
