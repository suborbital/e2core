package server

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/suite"
	"github.com/suborbital/vektor/vk"
	"github.com/suborbital/vektor/vtest"
	"github.com/suborbital/velocity/orchestrator"
	"github.com/suborbital/velocity/server/options"
	"github.com/suborbital/velocity/signaler"
)

type serverTestSuite struct {
	suite.Suite
	ts       *vk.Server
	o        *orchestrator.Orchestrator
	signaler *signaler.Signaler
	lock     sync.Mutex
}

// SetupSuite sets up the entire suite
func (s *serverTestSuite) SetupSuite() {
	fmt.Println("SETUP")
}

// TearDownSuite tears everything down
func (s *serverTestSuite) TearDownSuite() {
	fmt.Println("TEARDOWN")

	s.signaler.ManualShutdown(time.Second)
}

//curl -d 'my friend' localhost:8080/hello.
func (s *serverTestSuite) TestHelloEndpoint() {
	server, err := s.serverForBundle("../example-project/runnables.wasm.zip")
	if err != nil {
		s.T().Error(errors.Wrap(err, "failed to s.serverForBundle"))
		return
	}

	vt := vtest.New(server) //creating fake version of the server that we can send requests to and it will behave same was as if it was the real server.

	req, err := http.NewRequest(http.MethodPost, "/hello", bytes.NewBuffer([]byte("my friend")))
	if err != nil {
		s.T().Fatal(err)
	}

	vt.Do(req, s.T()).
		AssertStatus(200).
		AssertBodyString("hello my friend")
}

//curl -d 'name' localhost:8080/set/name
//curl localhost:8080/get/name.
func (s *serverTestSuite) TestSetAndGetKeyEndpoints() {
	server, err := s.serverForBundle("../example-project/runnables.wasm.zip")
	if err != nil {
		s.T().Error(errors.Wrap(err, "failed to s.serverForBundle"))
		return
	}

	vt := vtest.New(server)
	req, err := http.NewRequest(http.MethodPost, "/set/name", bytes.NewBuffer([]byte("Suborbital")))
	if err != nil {
		s.T().Fatal(err)
	}
	newreq, err := http.NewRequest(http.MethodGet, "/get/name", bytes.NewBuffer([]byte{}))
	if err != nil {
		s.T().Fatal(err)
	}

	vt.Do(req, s.T()).
		AssertStatus(200)
	vt.Do(newreq, s.T()).
		AssertStatus(200).
		AssertBodyString("Suborbital")

}

//curl localhost:8080/file/main.md.
func (s *serverTestSuite) TestFileMainMDEndpoint() {
	server, err := s.serverForBundle("../example-project/runnables.wasm.zip")
	if err != nil {
		s.T().Error(errors.Wrap(err, "failed to s.serverForBundle"))
		return
	}

	vt := vtest.New(server)
	req, err := http.NewRequest(http.MethodGet, "/file/main.md", bytes.NewBuffer([]byte{}))
	if err != nil {
		s.T().Fatal(err)
	}

	vt.Do(req, s.T()).
		AssertStatus(200).
		AssertBodyString("## hello")
}

//curl localhost:8080/file/css/main.css.
func (s *serverTestSuite) TestFileMainCSSEndpoint() {
	server, err := s.serverForBundle("../example-project/runnables.wasm.zip")
	if err != nil {
		s.T().Error(errors.Wrap(err, "failed to s.serverForBundle"))
		return
	}

	vt := vtest.New(server)
	req, err := http.NewRequest(http.MethodGet, "/file/css/main.css", bytes.NewBuffer([]byte{}))
	if err != nil {
		s.T().Fatal(err)
	}
	data, err := os.ReadFile("../example-project/static/css/main.css")
	if err != nil {
		s.T().Fatal(err)
	}

	vt.Do(req, s.T()).
		AssertStatus(200).
		AssertBody(data)
}

// curl localhost:8080/file/js/app/main.js.
func (s *serverTestSuite) TestFileMainJSEndpoint() {
	server, err := s.serverForBundle("../example-project/runnables.wasm.zip")
	if err != nil {
		s.T().Error(errors.Wrap(err, "failed to s.serverForBundle"))
		return
	}

	vt := vtest.New(server)
	req, err := http.NewRequest(http.MethodGet, "/file/js/app/main.js", bytes.NewBuffer([]byte{})) //change to struct initializer format byte{}.
	if err != nil {
		s.T().Fatal(err)
	}

	data, err := os.ReadFile("../example-project/static/js/app/main.js")
	if err != nil {
		s.T().Fatal(err)
	}

	vt.Do(req, s.T()).
		AssertStatus(200).
		AssertBody(data)
}

//curl -d 'https://github.com' localhost:8080/fetch | grep "grav".
func (s *serverTestSuite) TestFetchEndpoint() {
	server, err := s.serverForBundle("../example-project/runnables.wasm.zip")
	if err != nil {
		s.T().Error(errors.Wrap(err, "failed to s.serverForBundle"))
		return
	}

	vt := vtest.New(server)
	req, err := http.NewRequest(http.MethodPost, "/fetch", bytes.NewBuffer([]byte("https://github.com")))
	if err != nil {
		s.T().Fatal(err)
	}
	resp := vt.Do(req, s.T())

	// Check the response for these "Repositories", "People" and "Sponsoring" keywords to ensure that the correct HTML
	// has been loaded.
	ar := []string{
		"Repositories",
		"People",
		"Sponsoring",
	}

	s.T().Run("contains", func(t *testing.T) {
		for _, r := range ar {
			s.T().Run(r, func(t *testing.T) {
				if !strings.Contains(string(resp.Body), r) {
					s.T().Errorf("Couldn't find %s in the response", r)
				}
			})
		}
	})
}

// nolint
func (s *serverTestSuite) serverForBundle(filepath string) (*vk.Server, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.ts == nil {
		server, err := New(options.UseBundlePath(filepath))
		if err != nil {
			return nil, errors.Wrap(err, "failed to New")
		}

		testServer, err := server.testServer()
		if err != nil {
			return nil, errors.Wrap(err, "failed to s.testServer")
		}

		orchestrator, err := orchestrator.New(filepath)
		if err != nil {
			return nil, errors.Wrap(err, "failed to orchestrator.New")
		}

		signaler := signaler.Setup()

		signaler.Start(orchestrator.Start)

		time.Sleep(time.Second * 3)

		s.o = orchestrator
		s.ts = testServer
		s.signaler = signaler
	}

	return s.ts, nil
}

func TestServerTestSuite(t *testing.T) {
	suite.Run(t, new(serverTestSuite))
}