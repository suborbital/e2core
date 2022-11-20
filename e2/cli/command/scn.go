package command

import (
	"os"

	"github.com/suborbital/e2core/e2/scn"
)

func scnAPI() *scn.API {
	endpoint := scn.DefaultEndpoint
	if envEndpoint, exists := os.LookupEnv(scnEndpointEnvKey); exists {
		endpoint = envEndpoint
	}

	api := scn.New(endpoint)

	return api
}
