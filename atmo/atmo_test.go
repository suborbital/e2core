package atmo

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/suborbital/atmo/atmo/options"
	"github.com/suborbital/vektor/vtest"
)

//I think the easiest way would be to make a single Atmo method that does everything Start() does but instead looks something like func (a *Atmo) testStart() (*vk.Server, err), and then you can call that method from your atmo_test.go tests

//Create an Atmo object
// func New(opts ...options.Modifier) *Atmo {
// 	options.UseBundlePath(path),

// }

// Just call: New Method which creates a new atmo server and object

//and give it this option.

// UseBundlePath("./example-project/runnables.wasm.zip")
/*
Tests:
If any tests do not return 200s, fail the test (CoNNOR WILL SHOW ME)
RECREATE ALL TESTS IN GOLANG

1.
/hello       ****** curl -d 'my friend' localhost:8080/hello *********
Make a request to each of the handlpoints and check its results.
Sendpost request to /hello and should get "hello my friend" back (if you send my friend in post).


2.
/fetch: *******    curl -d 'https://github.com' localhost:8080/fetch | grep "atmo"    ********
Testing use of State- modifyurl function - adds /suborbital onto end of url and fetch test function takes that url and makes a request to that url.
Make request to /fetch and body of request will be HTTPS.github.com- modify url will change that to github.com/suborbital and fetchtest will make a request to github.com/suborbital and then it will downtload the html of our suborbital github page.
SO. POST REQUEST to /fetch
Body of post request should be HTTPS github.com
Response will be HTML, but search through the string to see if Vektor, Grav, and Atmo are there, that is the correct page.
The HTML will come back as a string.
Strings.contains() --> built in go function.  ......... (14.29)

3.
/set/:key and /get/:key

^^ to test: first request should have 200 response code and second has 200 and exact same value. /set/name and /get/name Botha re testing cache. (won't return any data for SET)- for check: ONLY checking that status code =200)
pass variable into :key (like /set/name - Connor for example, and then /get as its the same to


/set/:key *********      curl -d 'nyah' localhost:8080/set/name ***************.  Does not return anything- on a 200 response it doesn't return any day (this POSt request does not return data). Just check that the status code is 200.

/get/:key *********      curl localhost:8080/get/name    *********** Should return name from above.

4.
/file/*file - "testing fetching a file from the static directory" ****** curl localhost:8080/file/main.md ********* will return contents of main.md *****
RETURNS: [contents of main.md file] which is in this case: hello%

Or: [other file example]
curl localhost:8080/file/css/main.css
RETURNS:
.classname {
	color: aliceblue;
}%

make a request to each file in Static directory (/css/main.css, /js/app/main.js, /main.md) and should return contents of the files.
Compare return values of each return request to ensure it matches the contents of the file that you are asking for. Can jsut copy contents of file and read test or READ file (more automated).

* = entire path (like /file/{css/main.css}), : = one element (vs /file/{main.css}) _ you can only have one element at the latter
like: /file/css/main.css
You can peek at each runnable that each fo these is using just to see how the runnable is writtena nd what APIs its using for general knowledge about atmo.

/github: uses graphql function (LEAVE FOR NOW)- come back to because requires

Ignore: /user--> (requires db) and /stream --> (requires web socket).

*/
//curl -d 'my friend' localhost:8080/hello
//send post request to endpoint
//should return "hello my friend"

// 3 test important notes:
// Method has to start w Test405Request
// Param has to be (t *testing.T)
// file name has to be _test.go

// /hello       ****** curl -d 'my friend' localhost:8080/hello *********
func TestEchoRequest(t *testing.T) {
	atmo := atmoForBundle("../example-project/runnables.wasm.zip")

	server, err := atmo.testServer()
	if err != nil {
		t.Error(err)
		return
	}

	vt := vtest.New(server) //creating fake version of the server that we can send requests to and it will behave same was as if it was the real server

	req, _ := http.NewRequest(http.MethodPost, "/hello", bytes.NewBuffer([]byte("my friend"))) //same way to use curl
	// http.MethodGet.  file. bytes.NewBuffer([]byte("")//data from the body

	resp := vt.Do(req, t) //execute the request and returns the response  (can use resp to do the checks)

	resp.AssertStatus(200) //checks if response status code is 200 and if not it will fail
}

func TestSetKey(t *testing.T) {
	atmo := atmoForBundle("../example-project/runnables.wasm.zip")

	server, err := atmo.testServer()
	if err != nil {
		t.Error(err)
		return
	}

	vt := vtest.New(server)

	req, _ := http.NewRequest(http.MethodPost, "/set/name", bytes.NewBuffer([]byte("Suborbital"))) //same way to use curl

	resp := vt.Do(req, t)
	resp.AssertStatus(200) //checks if response status code is 200 and if not it will fail
}

func TestGetKeyRequest(t *testing.T) {
	atmo := atmoForBundle("../example-project/runnables.wasm.zip")

	server, err := atmo.testServer()
	if err != nil {
		t.Error(err)
		return
	}

	vt := vtest.New(server)
	req, _ := http.NewRequest(http.MethodPost, "/set/name", bytes.NewBuffer([]byte("Suborbital"))) //same way to use curl
	newreq, _ := http.NewRequest(http.MethodGet, "/get/name", bytes.NewBuffer([]byte("Suborbital")))

	vt.Do(req, t)
	getresp := vt.Do(newreq, t)
	getresp.AssertStatus(200)
	getresp.AssertBodyString("Suborbital")
}

// // curl -d 'https://github.com' localhost:8080/fetch | grep "atmo"
// func TestModifyUrlRequest(t *testing.T) {
// 	atmo := atmoForBundle("../example-project/runnables.wasm.zip")

// 	server, err := atmo.testServer()
// 	if err != nil {
// 		t.Error(err)
// 		return
// 	}

// 	vt := vtest.New(server)

// 	req, _ := http.NewRequest(http.MethodPost, "/", nil)

// 	resp := vt.Do(req, t)

// 	fmt.Println(string(resp.Body))

// 	resp.AssertStatus(200)
// 	resp.AssertBodyString("hello my friend")
// 	// resp := resp.
// 	// Strings.contains("atmo")
// 	// Strings.contains("grav")
// 	// Strings.contains("vektor")

// 	//resp.Body-- you can access for HTMl reading and pass this into string.contains (check name of function)
// }

//Instead Recreate calling it
func atmoForBundle(filepath string) *Atmo {
	return New(options.UseBundlePath(filepath))
}
