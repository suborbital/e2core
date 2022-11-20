package scn

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/pkg/errors"

	"github.com/suborbital/e2core/e2/scn/types"
)

const (
	DefaultEndpoint       = "https://api.suborbital.network"
	tokenRequestHeaderKey = "X-Suborbital-Env-Token"
)

// API is an API client.
type API struct {
	endpoint string
}

// VerifiedAPI is an API that has an email-verified access level.
type VerifiedAPI struct {
	api      *API
	verifier *types.RequestVerifier
}

// EnvironmentAPI is an API authenticated to a particular SCN environment.
type EnvironmentAPI struct {
	api   *API
	token string
}

func New(endpoint string) *API {
	s := &API{
		endpoint: endpoint,
	}

	return s
}

// ForVerifiedEmail verifies an email address is correct and then creates a VerifiedAPI object.
func (a *API) ForVerifiedEmail(email string, codeFn func() (string, error)) (*VerifiedAPI, error) {
	verifier, err := a.createEmailVerifier(email)
	if err != nil {
		return nil, errors.Wrap(err, "failed to createEmailVerifier")
	}

	code, err := codeFn()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get verifier code")
	}

	reqVerifier := &types.RequestVerifier{
		UUID: verifier.UUID,
		Code: code,
	}

	verified := &VerifiedAPI{
		api:      a,
		verifier: reqVerifier,
	}

	return verified, nil
}

// ForEnvironment returns an EnvironmentAPI scoped to the given token.
func (a *API) ForEnvironment(token string) (*EnvironmentAPI, error) {
	env := &EnvironmentAPI{
		api:   a,
		token: token,
	}

	return env, nil
}

// do performs a request.
func (a *API) do(method string, URI string, body, result interface{}) error {
	return a.doWithHeaders(method, URI, nil, body, result)
}

// doWithHeaders performs a request with the provided headers.
func (a *API) doWithHeaders(method string, URI string, headers map[string]string, body, result interface{}) error {
	var buffer io.Reader

	URL, err := url.Parse(fmt.Sprintf("%s%s", a.endpoint, URI))
	if err != nil {
		return errors.Wrap(err, "faield to parse URL")
	}

	if body != nil {
		bodyJSON, err := json.Marshal(body)
		if err != nil {
			return errors.Wrap(err, "failed to Marshal")
		}

		buffer = bytes.NewBuffer(bodyJSON)
	}

	request, err := http.NewRequest(method, URL.String(), buffer)
	if err != nil {
		return errors.Wrap(err, "failed to NewRequest")
	}

	if headers != nil {
		reqHeader := request.Header
		for k, v := range headers {
			reqHeader.Set(k, v)
		}

		request.Header = reqHeader
	}

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return errors.Wrap(err, "failed to Do request")
	}

	if resp.StatusCode > 299 {
		return fmt.Errorf("failed to Do request, received status code %d", resp.StatusCode)
	}

	if result != nil {
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return errors.Wrap(err, "failed to ReadAll body")
		}

		if err := json.Unmarshal(body, result); err != nil {
			return errors.Wrap(err, "failed to Unmarshal body")
		}
	}

	return nil
}
