package server

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/suite"

	"github.com/suborbital/appspec/system/bundle"
	"github.com/suborbital/e2core/e2core/satbackend"
	"github.com/suborbital/e2core/options"
	"github.com/suborbital/e2core/signaler"
	"github.com/suborbital/e2core/syncer"
	"github.com/suborbital/vektor/vk"
	"github.com/suborbital/vektor/vlog"
)

type serverTestSuite struct {
	suite.Suite
	ts       *vk.Server
	o        *satbackend.Orchestrator
	signaler *signaler.Signaler
	lock     sync.Mutex

	testStart time.Time
	shouldRun bool
}

// HandleStats will write a nice summary at the end after the teardown function.
func (s *serverTestSuite) NOHandleStats(suiteName string, stats *suite.SuiteInformation) {
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

	s.shouldRun = true
}

func (s *serverTestSuite) AfterTest(_, testName string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.ts != nil {
		err := s.ts.Stop()
		if err != nil {
			s.T().Logf("%s: shutting down server failed: %s", testName, err.Error())
		}

		s.ts = nil
	}

	if s.signaler != nil {
		err := s.signaler.ManualShutdown(time.Second)
		if err != nil {
			s.T().Logf("%s: shutting down signaler failed: %s", testName, err.Error())
		}
		s.signaler = nil
	}
}

// curl -d 'my friend' localhost:8080/hello.
func (s *serverTestSuite) TestHelloEndpoint() {
	if !s.shouldRun {
		s.T().Skip("Skipping")
	}

	err := s.serverForBundle("../example-project/modules.wasm.zip")
	s.Require().NoError(err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/name/com.suborbital.app/default/helloworld-rs", bytes.NewBuffer([]byte("my friend")))

	time.Sleep(5 * time.Second)

	s.ts.ServeHTTP(w, req)

	resultBody, err := io.ReadAll(w.Result().Body)
	s.Require().NoError(err)

	s.Equal(http.StatusOK, w.Result().StatusCode)
	s.Equal([]byte(`hello my friend`), resultBody)
}

// curl -d 'name' localhost:8080/set/name
// curl localhost:8080/get/name.
func (s *serverTestSuite) TestSetAndGetKeyEndpoints() {
	if !s.shouldRun {
		s.T().Skip("Skipping")
	}

	err := s.serverForBundle("../example-project/modules.wasm.zip")
	s.Require().NoError(err)

	setW := httptest.NewRecorder()
	getW := httptest.NewRecorder()

	setReq := httptest.NewRequest(http.MethodPost, "/name/com.suborbital.app/default/cache-set", bytes.NewBuffer([]byte("Suborbital")))
	getReq := httptest.NewRequest(http.MethodPost, "/name/com.suborbital.app/default/cache-get", bytes.NewBuffer(nil))

	s.ts.ServeHTTP(setW, setReq)
	s.Equal(http.StatusOK, setW.Result().StatusCode)

	s.ts.ServeHTTP(getW, getReq)
	s.Equal(http.StatusOK, getW.Result().StatusCode)

	// TODO: add central cache to get this test passing: https://github.com/suborbital/e2core/issues/238
	// AssertBodyString("Suborbital")
}

// curl localhost:8080/file/main.md.
func (s *serverTestSuite) TestFileMainMDEndpoint() {
	if !s.shouldRun {
		s.T().Skip("Skipping")
	}

	err := s.serverForBundle("../example-project/modules.wasm.zip")
	s.Require().NoError(err)

	w := httptest.NewRecorder()

	req := httptest.NewRequest(http.MethodPost, "/name/com.suborbital.app/default/get-file", bytes.NewBuffer(nil))

	req.Header.Add("X-Suborbital-State", `{"file": "main.md"}`)

	s.ts.ServeHTTP(w, req)

	responseBody, err := io.ReadAll(w.Result().Body)
	s.Require().NoError(err)

	s.Equal(http.StatusOK, w.Result().StatusCode)
	s.Equal([]byte(`## hello`), responseBody)
}

// curl localhost:8080/file/css/main.css.
func (s *serverTestSuite) TestFileMainCSSEndpoint() {
	if !s.shouldRun {
		s.T().Skip("Skipping")
	}

	err := s.serverForBundle("../example-project/modules.wasm.zip")
	s.Require().NoError(err, "error from serverForBundle for example project/modules.wasm.zip: %s")

	w := httptest.NewRecorder()

	req := httptest.NewRequest(http.MethodPost, "/name/com.suborbital.app/default/get-file", bytes.NewBuffer(nil))

	req.Header.Add("X-Suborbital-State", `{"file": "css/main.css"}`)

	data, err := os.ReadFile("../example-project/static/css/main.css")
	s.Require().NoError(err)

	s.ts.ServeHTTP(w, req)

	responseBody, err := io.ReadAll(w.Result().Body)
	s.Require().NoError(err)

	s.Equal(http.StatusOK, w.Result().StatusCode)
	s.Equal(data, responseBody)
}

// curl localhost:8080/file/js/app/main.js.
func (s *serverTestSuite) TestFileMainJSEndpoint() {
	if !s.shouldRun {
		s.T().Skip("Skipping")
	}

	err := s.serverForBundle("../example-project/modules.wasm.zip")
	s.Require().NoError(err)

	w := httptest.NewRecorder()

	req := httptest.NewRequest(http.MethodPost, "/name/com.suborbital.app/default/get-file", bytes.NewBuffer(nil))

	req.Header.Add("X-Suborbital-State", `{"file": "js/app/main.js"}`)

	data, err := os.ReadFile("../example-project/static/js/app/main.js")
	s.Require().NoError(err)

	s.ts.ServeHTTP(w, req)

	responseBody, err := io.ReadAll(w.Result().Body)
	s.Require().NoError(err)

	s.Equal(http.StatusOK, w.Result().StatusCode)
	s.Equal(data, responseBody)
}

// curl -d 'https://github.com' localhost:8080/fetch | grep "grav".
func (s *serverTestSuite) TestFetchEndpoint() {
	if !s.shouldRun {
		s.T().Skip("Skipping")
	}

	err := s.serverForBundle("../example-project/modules.wasm.zip")
	s.Require().NoError(err)

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

// nolint
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

	testServer, err := server.testServer()
	if err != nil {
		return errors.Wrap(err, "failed to s.testServer")
	}

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
