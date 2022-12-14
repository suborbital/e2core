package server

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/suite"

	"github.com/suborbital/e2core/e2core/backend/satbackend"
	"github.com/suborbital/e2core/e2core/options"
	"github.com/suborbital/e2core/e2core/syncer"
	"github.com/suborbital/e2core/foundation/signaler"
	"github.com/suborbital/systemspec/system/bundle"
	"github.com/suborbital/vektor/vk"
	"github.com/suborbital/vektor/vlog"
)

type serverTestSuite struct {
	suite.Suite
	ts       *vk.Server
	o        *satbackend.Orchestrator
	signaler *signaler.Signaler
	lock     sync.Mutex

	shouldRun bool
}

// HandleStats will write a nice summary at the end after the teardown function.
func (s *serverTestSuite) HandleStats(suiteName string, stats *suite.SuiteInformation) {
	s.T().Logf("Stats for suite '%s' ran in %s", suiteName, stats.End.Sub(stats.Start))
	verdict := ""

	s.T().Logf("length of the teststats: %d", len(stats.TestStats))

	for testName, info := range stats.TestStats {
		verdict = "FAIL"
		if info.Passed {
			verdict = "PASS"
		}
		s.T().Logf("%s -- %s ran in %s", verdict, testName, info.End.Sub(info.Start))
	}
}

// SetupSuite sets up the entire suite
func (s *serverTestSuite) SetupSuite() {
	if shouldRun := os.Getenv("RUN_SERVER_TESTS"); shouldRun == "true" {
		s.T().Logf("Suite Setup: Server tests will be run")
		s.shouldRun = true
	} else {
		s.T().Log("Suite Setup: Server tests will not be run")
	}

	err := s.serverForBundle("../../example-project/modules.wasm.zip")
	s.Require().NoError(err)

	err = s.ts.TestStart()
	s.Require().NoError(err)
}

func (s *serverTestSuite) TearDownSuite() {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.o != nil {
		s.T().Log("starting shutdown of orchestrator")
		s.o.Shutdown()
		s.T().Log("shutdown completed of orchestrator")

		s.o = nil
	}

	if s.signaler != nil {
		s.T().Log("starting shutdown of signaler")

		err := s.signaler.ManualShutdown(time.Second)
		s.Require().NoError(err)

		s.T().Log("shutdown completed of signaler")

		s.signaler = nil
	}

	if s.ts != nil {
		s.T().Log("starting shutdown of test server")

		ctx, cxl := context.WithTimeout(context.Background(), time.Second)
		defer cxl()
		err := s.ts.StopCtx(ctx)
		s.Require().NoError(err)

		s.T().Log("shutdown of test server completed")
	}

	time.Sleep(3 * time.Second)
}

func (s *serverTestSuite) AfterTest(_, testName string) {
	s.T().Logf("%s finished running", testName)
}

// curl -d 'my friend' localhost:8080/hello.
func (s *serverTestSuite) TestHelloEndpoint() {
	if !s.shouldRun {
		s.T().Skip("Skipping")
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/name/com.suborbital.app/default/helloworld-rs", bytes.NewBuffer([]byte("my friend")))

	s.ts.ServeHTTP(w, req)

	resultBody, err := io.ReadAll(w.Result().Body)
	s.Require().NoError(err)

	s.Equal(http.StatusOK, w.Result().StatusCode)
	s.Equal([]byte(`hello my friend`), resultBody)
}

// curl -d 'https://github.com' localhost:8080/fetch | grep "grav".
func (s *serverTestSuite) TestFetchEndpoint() {
	if !s.shouldRun {
		s.T().Skip("Skipping")
	}

	w := httptest.NewRecorder()

	req := httptest.NewRequest(http.MethodPost, "/workflow/com.suborbital.app/default/fetch", bytes.NewBuffer([]byte("https://github.com")))

	s.ts.ServeHTTP(w, req)

	responseBody, err := io.ReadAll(w.Result().Body)
	s.Require().NoError(err)

	bodyString := string(responseBody)

	// Check the response for these "Repositories", "People" and "Sponsoring" keywords to ensure that the correct HTML
	// has been loaded.
	ar := []string{
		"Repositories",
		"People",
		"Sponsoring",
	}

	for _, r := range ar {
		s.Containsf(bodyString, r, "responsebody (%s) did not contain string (%s)", responseBody, r)
	}
}

// serverForBundle creates a new test server based on the module reachable with filepath, assigns it to a struct level
// unexported property (s.ts), and starts it.
//
// To tear down the server we use the AfterTest(suiteName, testName string) method where we still have access to the
// server that's running currently.
func (s *serverTestSuite) serverForBundle(filepath string) error {
	if !s.shouldRun {
		s.T().Skip("Skipping")
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	logger := vlog.Default(vlog.Level(vlog.LogLevelInfo))

	opts := options.NewWithModifiers(options.UseBundlePath(filepath), options.UseLogger(logger))

	source := bundle.NewBundleSource(opts.BundlePath)

	syncR := syncer.New(opts, source)

	server, err := New(syncR, opts)
	if err != nil {
		return errors.Wrap(err, "failed to New")
	}

	testServer := server.testServer()

	orchestrator, err := satbackend.New(opts, syncR)
	if err != nil {
		return errors.Wrap(err, "failed to orchestrator.New")
	}

	sig := signaler.Setup()
	sig.Start(orchestrator.Start)

	time.Sleep(time.Second * 3)

	s.o = orchestrator
	s.ts = testServer
	s.signaler = sig

	return nil
}

func TestServerTestSuite(t *testing.T) {
	suite.Run(t, new(serverTestSuite))
}
