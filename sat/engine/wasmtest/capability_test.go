package wasmtest

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/suborbital/e2core/foundation/scheduler"
	"github.com/suborbital/e2core/sat/api"
	"github.com/suborbital/e2core/sat/engine"
	"github.com/suborbital/systemspec/capabilities"
)

func TestDisabledHTTP(t *testing.T) {
	config := capabilities.DefaultCapabilityConfig()
	config.HTTP = &capabilities.HTTPConfig{Enabled: false}

	api, _ := api.NewWithConfig(config)

	e := engine.NewWithAPI(api)

	// test a WASM module that is loaded directly instead of through the bundle
	doWasm, err := e.RegisterFromFile("wasm", "../testdata/fetch/fetch.wasm")
	require.NoError(t, err, "registerfrom file failed for fetch.wasm")

	_, err = doWasm("https://1password.com").Then()
	if err != nil {
		if err.Error() != `{"code":1,"message":"capability is not enabled"}` {
			t.Error("received incorrect error", err.Error())
		}
	} else {
		t.Error("runnable should have failed")
	}
}

func TestDisabledGraphQL(t *testing.T) {
	// bail out if GitHub auth is not set up (i.e. in Travis)
	// we want the Runnable to fail because graphql is disabled,
	// not because auth isn't set up correctly
	if _, ok := os.LookupEnv("GITHUB_TOKEN"); !ok {
		return
	}

	config := capabilities.DefaultCapabilityConfig()
	config.GraphQL = &capabilities.GraphQLConfig{Enabled: false}
	config.Auth = &capabilities.AuthConfig{
		Enabled: true,
		Headers: map[string]capabilities.AuthHeader{
			"api.github.com": {
				HeaderType: "bearer",
				Value:      "env(GITHUB_TOKEN)",
			},
		},
	}

	api, _ := api.NewWithConfig(config)

	e := engine.NewWithAPI(api)

	_, err := e.RegisterFromFile("rs-graphql", "../testdata/rs-graphql/rs-graphql.wasm")
	require.NoError(t, err, "registerfrom file failed for rs-graphql.wasm")

	_, err = e.Do(scheduler.NewJob("rs-graphql", nil)).Then()
	if err != nil {
		if err.Error() != `{"code":1,"message":"capability is not enabled"}` {
			t.Error("received incorrect error ", err.Error())
		}
	} else {
		t.Error("runnable should have produced an error")
	}
}
