//go:build integration
// +build integration

package tests

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

const e2corePort = 53091

// ConstDIntegrationSuite will test @todo complete this.
type ConstDIntegrationSuite struct {
	cmd    *exec.Cmd
	cancel context.CancelFunc

	suite.Suite
}

// TestConstDIntegrationSuite gets run from go's test framework that kicks off the suite.
func TestConstDIntegrationSuite(t *testing.T) {
	suite.Run(t, new(ConstDIntegrationSuite))
}

// SetupSuite runs first in the chain. Used to set up properties and settings
// that all test methods need access to.
func (i *ConstDIntegrationSuite) SetupSuite() {
	dir, err := os.Getwd()
	if err != nil {
		i.FailNowf("getwd", "get working directory: %s", err.Error())
	}

	constdWorkingDir := filepath.Join(dir, "../constd/example-project")
	exampleZip := filepath.Join(constdWorkingDir, "runnables.wasm.zip")
	constdExecPath := filepath.Join(dir, "../.bin/constd")

	ctx, cancel := context.WithCancel(context.Background())
	i.cancel = cancel

	i.cmd = exec.CommandContext(ctx, constdExecPath, exampleZip)
	i.cmd.Stdout = os.Stdout
	i.cmd.Stderr = os.Stdout
	i.cmd.Env = append(os.Environ(), fmt.Sprintf("CONSTD_ATMO_PORT=%d", e2corePort), "CONSTD_EXEC_MODE=metal")

	err = i.cmd.Start()
	i.Require().NoError(err)

	<-time.After(3 * time.Second)
}

// TearDownSuite runs last, and is usually used to close database connections
// or clear up after running the suite.
func (i *ConstDIntegrationSuite) TearDownSuite() {
	err := i.cmd.Process.Signal(syscall.SIGINT)
	i.Require().NoError(err)

	err = i.cmd.Wait()
	i.Require().NoError(err)
}

// TestSatEndpoints is an example test method. Any method that starts with Test* is
// going to be run. The test methods should be independent of each other and
// their order of execution should not matter, and you should also be able to
// run an individual test method on the suite without any issues.
func (i *ConstDIntegrationSuite) TestSatEndpoints() {
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
			name:                "constd runs hello echo successfully",
			path:                "/hello",
			requestVerb:         http.MethodPost,
			payload:             []byte(`bob`),
			wantStatus:          http.StatusOK,
			wantResponsePayload: []byte(`hello bob`),
		},
	}

	client := http.Client{
		Timeout: 2 * time.Second,
	}

	baseUrl := fmt.Sprintf("http://localhost:%d", e2corePort)

	for _, tCase := range tcs {
		i.Run(tCase.name, func() {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			req, err := http.NewRequestWithContext(ctx, tCase.requestVerb, baseUrl+tCase.path, bytes.NewReader(tCase.payload))
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
