package server

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/suite"

	"github.com/suborbital/e2core/e2core/backend/satbackend"
	"github.com/suborbital/e2core/e2core/options"
	"github.com/suborbital/e2core/e2core/syncer"
	"github.com/suborbital/systemspec/system/bundle"
)

type serverTestSuite struct {
	suite.Suite
	ts   *echo.Echo
	o    *satbackend.Orchestrator
	lock sync.Mutex

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

	if s.ts != nil {
		s.T().Log("starting shutdown of test server")

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

	logger := zerolog.New(os.Stderr).With().Timestamp().Str("service", "serverForBundle test").Logger()

	opts, err := options.NewWithModifiers(options.UseBundlePath(filepath))
	s.Require().NoError(err, "options.NewWithModifiers")

	source := bundle.NewBundleSource(opts.BundlePath)

	syncR := syncer.New(opts, logger, source)

	server, err := New(logger, syncR, opts)
	if err != nil {
		return errors.Wrap(err, "failed to New")
	}

	testServer := server.testServer()

	orchestrator, err := satbackend.New(logger, opts, syncR)
	if err != nil {
		return errors.Wrap(err, "failed to orchestrator.New")
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	serverErrors := make(chan error, 1)

	go func() {
		logger.Info().Msg("starting server")
		err := orchestrator.Start()
		if err != nil {
			serverErrors <- errors.Wrap(err, "orchestrator.Start")
		}
	}()

	time.Sleep(time.Second * 3)

	s.o = orchestrator
	s.ts = testServer

	return nil
}

func TestServerTestSuite(t *testing.T) {
	suite.Run(t, new(serverTestSuite))
}
