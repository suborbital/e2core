package atmo

import (
	"bytes"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/suborbital/atmo/atmo/options"
	"github.com/suborbital/vektor/vtest"
)

//curl -d 'my friend' localhost:8080/hello
func TestHelloEndpoint(t *testing.T) {
	atmo := atmoForBundle("../example-project/runnables.wasm.zip")

	server, err := atmo.testServer()
	if err != nil {
		t.Error(err)
		return
	}

	vt := vtest.New(server) //creating fake version of the server that we can send requests to and it will behave same was as if it was the real server

	req, _ := http.NewRequest(http.MethodPost, "/hello", bytes.NewBuffer([]byte("my friend")))

	vt.Do(req, t).
		AssertStatus(200).
		AssertBodyString("hello my friend")
}

//curl -d 'name' localhost:8080/set/name
//curl localhost:8080/get/name
func TestSetAndGetKeyEndpoints(t *testing.T) {
	atmo := atmoForBundle("../example-project/runnables.wasm.zip")

	server, err := atmo.testServer()
	if err != nil {
		t.Error(err)
		return
	}

	vt := vtest.New(server)
	req, _ := http.NewRequest(http.MethodPost, "/set/name", bytes.NewBuffer([]byte("Suborbital")))
	newreq, _ := http.NewRequest(http.MethodGet, "/get/name", bytes.NewBuffer([]byte("Suborbital")))

	vt.Do(req, t).
		AssertStatus(200)
	vt.Do(newreq, t).
		AssertStatus(200).
		AssertBodyString("Suborbital")

}

//curl localhost:8080/file/main.md
func TestFileMainMDEndpoint(t *testing.T) {
	atmo := atmoForBundle("../example-project/runnables.wasm.zip")

	server, err := atmo.testServer()
	if err != nil {
		t.Error(err)
		return
	}

	vt := vtest.New(server)
	req, _ := http.NewRequest(http.MethodGet, "/file/main.md", bytes.NewBuffer([]byte("")))

	vt.Do(req, t).
		AssertStatus(200).
		AssertBodyString("## hello")
}

//curl localhost:8080/file/css/main.css
func TestFileMainCSSEndpoint(t *testing.T) {
	atmo := atmoForBundle("../example-project/runnables.wasm.zip")

	server, err := atmo.testServer()
	if err != nil {
		t.Error(err)
		return
	}

	vt := vtest.New(server)
	req, _ := http.NewRequest(http.MethodGet, "/file/css/main.css", bytes.NewBuffer([]byte("")))

	data, err := os.ReadFile("../example-project/static/css/main.css")
	if err != nil {
		t.Error(err)
	}

	vt.Do(req, t).
		AssertStatus(200).
		AssertBody(data)
}

// curl localhost:8080/file/js/app/main.js
func TestFileMainJSEndpoint(t *testing.T) {
	atmo := atmoForBundle("../example-project/runnables.wasm.zip")

	server, err := atmo.testServer()
	if err != nil {
		t.Error(err)
		return
	}

	vt := vtest.New(server)
	req, _ := http.NewRequest(http.MethodGet, "/file/js/app/main.js", bytes.NewBuffer([]byte("")))

	data, err := os.ReadFile("../example-project/static/js/app/main.js")
	if err != nil {
		t.Error(err)
	}

	vt.Do(req, t).
		AssertStatus(200).
		AssertBody(data)
}

//curl -d 'https://github.com' localhost:8080/fetch | grep "grav"
func TestFetchEndpoint(t *testing.T) {
	atmo := atmoForBundle("../example-project/runnables.wasm.zip")

	server, err := atmo.testServer()
	if err != nil {
		t.Error(err)
		return
	}

	vt := vtest.New(server)
	req, _ := http.NewRequest(http.MethodPost, "/fetch", bytes.NewBuffer([]byte("https://github.com")))
	resp := vt.Do(req, t)

	//check the response for these "atmo", "grav" and "vektor" keywords to ensure that the correct HTML has been loaded
	ar := []string{
		"atmo",
		"grav",
		"vektor",
	}

	t.Run("contains", func(t *testing.T) {
		for _, s := range ar {
			t.Run(s, func(t *testing.T) {
				if !strings.Contains(string(resp.Body), s) {
					t.Errorf("Couldn't find, %s in the response", s)
				}
			})
		}
	})
}

func atmoForBundle(filepath string) *Atmo {
	return New(options.UseBundlePath(filepath))
}
