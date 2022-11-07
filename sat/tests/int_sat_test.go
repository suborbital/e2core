//go:build integration
// +build integration

package tests

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// IntegrationSuite will test @todo complete this.
type IntegrationSuite struct {
	suite.Suite

	container tc.Container
	ctxCloser context.CancelFunc
}

// TestIntegrationSuite gets run from go's test framework that kicks off the suite.
func pppTestIntegrationSuite(t *testing.T) {
	suite.Run(t, new(IntegrationSuite))
}

// SetupSuite runs first in the chain. Used to set up properties and settings
// that all test methods need access to.
func (i *IntegrationSuite) SetupSuite() {
	dir, err := os.Getwd()
	if err != nil {
		i.FailNowf("getwd", "what: %s", err.Error())
	}

	satWorkingDir := filepath.Join(dir, "../examples")
	ctx, ctxCloser := context.WithTimeout(context.Background(), 10*time.Second)
	defer ctxCloser()

	req := tc.GenericContainerRequest{
		ContainerRequest: tc.ContainerRequest{
			Image: "suborbital/sat:dev",
			Env: map[string]string{
				"SAT_HTTP_PORT": "8080",
			},
			ExposedPorts: []string{"8080:8080"},
			Cmd: []string{
				"sat", "/runnables/hello-echo/hello-echo.wasm",
			},
			Mounts: []tc.ContainerMount{
				{
					Source: tc.DockerBindMountSource{
						HostPath: satWorkingDir,
					},
					Target: tc.ContainerMountTarget("/runnables"),
				},
			},
			AutoRemove: true,
			WaitingFor: wait.NewHTTPStrategy("/").WithPort("8080/tcp").WithMethod(http.MethodPost).WithBody(bytes.NewBuffer([]byte(`hi`))),
		},
		Started:      true,
		ProviderType: tc.ProviderDocker,
	}

	container, err := tc.GenericContainer(ctx, req)
	i.Require().NoError(err)

	i.container = container
}

// TearDownSuite runs last, and is usually used to close database connections
// or clear up after running the suite.
func (i *IntegrationSuite) TearDownSuite() {
	terminateCtx, termCtxCloser := context.WithTimeout(context.Background(), 3*time.Second)
	defer termCtxCloser()

	tearDownChan := make(chan struct{}, 1)

	// set up teardown before we terminate the container, because if we terminate first and then set up this exit
	// strategy after, depending on the computer the container might have been torn down already, which will cause the
	// strategy to panic due to a nil pointer, because there's no .State() on a torn down (nil) container.
	go func() {
		err := wait.NewExitStrategy().
			WithPollInterval(3*time.Second).
			WithExitTimeout(2*time.Minute).
			WaitUntilReady(context.Background(), i.container)
		i.Require().NoError(err)

		tearDownChan <- struct{}{}
	}()

	err := i.container.Terminate(terminateCtx)
	i.Require().NoError(err)

	<-tearDownChan
}

// TestSatEndpoints is an example test method. Any method that starts with Test* is
// going to be run. The test methods should be independent of each other and
// their order of execution should not matter, and you should also be able to
// run an individual test method on the suite without any issues.
func (i *IntegrationSuite) TestSatEndpoints() {
	type testCase struct {
		name                string
		path                string
		requestVerb         string
		payload             []byte
		wantStatus          int
		wantResponsePayload []byte
	}

	tcs := []testCase{
		{
			name:                "sat runs hello echo successfully",
			path:                "",
			requestVerb:         http.MethodPost,
			payload:             []byte(`{"text":"from Bob Morane"}`),
			wantStatus:          http.StatusOK,
			wantResponsePayload: []byte(`hello {"text":"from Bob Morane"}`),
		},
	}

	client := http.Client{
		Timeout: 2 * time.Second,
	}

	baseUrl := "http://localhost:8080"

	for _, tCase := range tcs {
		i.Run(tCase.name, func() {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()
			req, err := http.NewRequestWithContext(ctx, tCase.requestVerb, baseUrl+"/"+tCase.path, bytes.NewReader(tCase.payload))
			i.Require().NoError(err)
			resp, err := client.Do(req)
			i.Require().NoError(err)

			responseBody, err := ioutil.ReadAll(resp.Body)
			i.Require().NoError(err)

			i.Equal(tCase.wantStatus, resp.StatusCode)
			i.Equal(tCase.wantResponsePayload, responseBody)
		})
	}
}
