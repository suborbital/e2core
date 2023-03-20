package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"

	"github.com/suborbital/e2core/e2core/options"
	"github.com/suborbital/systemspec/system"
)

var ErrUnauthorized = errors.New("received 401 Unauthorized from API")

// NewApiAuthClient returns a configured SE2 Auth client that will ask the tenant endpoint for info about a tenant by
// its ID.
func NewApiAuthClient(opts *options.Options) *APIAuthorizer {
	return &APIAuthorizer{
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
		location: opts.ControlPlane + "/environment/v1/tenant/%s",
	}
}

// APIAuthorizer is a bridge between e2core and a REST API. The only thing this one does is asks SE2 information about a
// tenant identified by the identifier over plain old REST API.
type APIAuthorizer struct {
	location   string
	httpClient *http.Client
}

// Authorize implements the Authorizer interface. It doesn't use the namespace and the function name arguments as its
// only job is to use the token it received and the identifier and send a message to SE2 asking info about the tenant.
//
// SE2, in turn, is the system that actually does the check for the token.
func (client *APIAuthorizer) Authorize(token system.Credential, identifier, _, _ string) (TenantInfo, error) {
	if token == nil {
		return TenantInfo{}, errors.New("no credentials provided")
	}

	authzReq, err := http.NewRequest(http.MethodGet, fmt.Sprintf(client.location, identifier), nil)
	if err != nil {
		return TenantInfo{}, errors.Wrap(err, "http.NewRequest GET control plane environment tenant")
	}

	// pass token along
	authzReq.Header.Set("Authorization", fmt.Sprintf("%s %s", token.Scheme(), token.Value()))

	resp, err := client.httpClient.Do(authzReq)
	if err != nil {
		return TenantInfo{}, errors.Wrap(err, "client.httpClient.Do")
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return TenantInfo{}, ErrUnauthorized
	}

	if resp.StatusCode != http.StatusOK {
		return TenantInfo{}, fmt.Errorf("received non-200 and non-401 status code %d from API", resp.StatusCode)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	var claims TenantInfo
	if err = json.NewDecoder(resp.Body).Decode(&claims); err != nil {
		return TenantInfo{}, errors.Wrap(err, "json.Decode into TenantInfo")
	}

	return claims, nil
}
