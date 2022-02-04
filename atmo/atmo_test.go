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

func TestHelloEndpoint(t *testing.T) {
	atmo := atmoForBundle("../example-project/runnables.wasm.zip")

	server, err := atmo.testServer()
	if err != nil {
		t.Error(err)
		return
	}

	vt := vtest.New(server) //creating fake version of the server that we can send requests to and it will behave same was as if it was the real server

	req, _ := http.NewRequest(http.MethodPost, "/hello", bytes.NewBuffer([]byte("my friend"))) //same way to use curl

	resp := vt.Do(req, t) //execute the request and returns the response  (can use resp to do the checks)

	resp.AssertStatus(200) //checks if response status code is 200 and if not it will fail
	resp.AssertBodyString("hello my friend")
}

func TestSetAndGetKeyEndpoints(t *testing.T) {
	atmo := atmoForBundle("../example-project/runnables.wasm.zip")

	server, err := atmo.testServer()
	if err != nil {
		t.Error(err)
		return
	}

	vt := vtest.New(server)
	req, _ := http.NewRequest(http.MethodPost, "/set/name", bytes.NewBuffer([]byte("Suborbital"))) //same way to use curl
	newreq, _ := http.NewRequest(http.MethodGet, "/get/name", bytes.NewBuffer([]byte("Suborbital")))

	resp := vt.Do(req, t)
	getresp := vt.Do(newreq, t)
	resp.AssertStatus(200)
	getresp.AssertStatus(200)
	getresp.AssertBodyString("Suborbital")
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
	req, _ := http.NewRequest(http.MethodGet, "/file/main.md", bytes.NewBuffer([]byte(""))) //same way to use curl

	resp := vt.Do(req, t)
	resp.AssertStatus(200)
	resp.AssertBodyString("## hello")
}

func TestFileMainCSSEndpoint(t *testing.T) {
	atmo := atmoForBundle("../example-project/runnables.wasm.zip")

	server, err := atmo.testServer()
	if err != nil {
		t.Error(err)
		return
	}

	vt := vtest.New(server)
	req, _ := http.NewRequest(http.MethodGet, "/file/css/main.css", bytes.NewBuffer([]byte(""))) //same way to use curl

	resp := vt.Do(req, t)
	resp.AssertStatus(200)

	data, err := os.ReadFile("../example-project/static/css/main.css")
	if err != nil {
		t.Error(err)
	}

	resp.AssertBody(data)
}

func TestFileMainJSEndpoint(t *testing.T) {
	atmo := atmoForBundle("../example-project/runnables.wasm.zip")

	server, err := atmo.testServer()
	if err != nil {
		t.Error(err)
		return
	}

	vt := vtest.New(server)
	req, _ := http.NewRequest(http.MethodGet, "/file/js/app/main.js", bytes.NewBuffer([]byte(""))) //same way to use curl

	resp := vt.Do(req, t)
	resp.AssertStatus(200)
	data, err := os.ReadFile("../example-project/static/js/app/main.js")
	if err != nil {
		t.Error(err)
	}

	resp.AssertBody(data)
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
		for _, s := range ar { //first v is the element of the array, and second v is array itself
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
