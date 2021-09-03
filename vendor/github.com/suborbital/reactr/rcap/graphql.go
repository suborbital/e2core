package rcap

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
)

// GraphQLConfig is configuration for the GraphQL capability
type GraphQLConfig struct {
	Enabled bool      `json:"enabled" yaml:"enabled"`
	Rules   HTTPRules `json:"rules" yaml:"rules"`
}

// GraphQLCapability is a GraphQL capability for Reactr Modules
type GraphQLCapability interface {
	Do(auth AuthCapability, endpoint, query string) (*GraphQLResponse, error)
}

// defaultGraphQLClient is the default implementation of the GraphQL capability
type defaultGraphQLClient struct {
	config GraphQLConfig
	client *http.Client
}

// DefaultGraphQLClient creates a GraphQLClient object
func DefaultGraphQLClient(config GraphQLConfig) GraphQLCapability {
	g := &defaultGraphQLClient{
		config: config,
		client: http.DefaultClient,
	}

	return g
}

// GraphQLRequest is a request to a GraphQL endpoint
type GraphQLRequest struct {
	Query         string            `json:"query"`
	Variables     map[string]string `json:"variables,omitempty"`
	OperationName string            `json:"operationName,omitempty"`
}

// GraphQLResponse is a GraphQL response
type GraphQLResponse struct {
	Data   map[string]interface{} `json:"data"`
	Errors []GraphQLError         `json:"errors,omitempty"`
}

// GraphQLError is a GraphQL error
type GraphQLError struct {
	Message string `json:"message"`
	Path    string `json:"path"`
}

func (g *defaultGraphQLClient) Do(auth AuthCapability, endpoint, query string) (*GraphQLResponse, error) {
	if !g.config.Enabled {
		return nil, ErrCapabilityNotEnabled
	}

	r := &GraphQLRequest{
		Query:     query,
		Variables: map[string]string{},
	}

	reqBytes, err := json.Marshal(r)
	if err != nil {
		return nil, errors.Wrap(err, "failed to Marshal request")
	}

	endpointURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, errors.Wrap(err, "failed to Parse endpoint")
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewBuffer(reqBytes))
	if err != nil {
		return nil, errors.Wrap(err, "failed to NewRequest")
	}

	if err := g.config.Rules.requestIsAllowed(req); err != nil {
		return nil, errors.Wrap(err, "failed to requestIsAllowed")
	}

	req.Header.Add("Content-Type", "application/json")

	authHeader := auth.HeaderForDomain(endpointURL.Host)
	if authHeader != nil && authHeader.Value != "" {
		req.Header.Add("Authorization", fmt.Sprintf("%s %s", authHeader.HeaderType, authHeader.Value))
	}

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to Do")
	}

	defer resp.Body.Close()

	respJSON, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to ReadAll body")
	}

	gqlResp := &GraphQLResponse{}
	if err := json.Unmarshal(respJSON, gqlResp); err != nil {
		return nil, errors.Wrap(err, "failed to Unmarshal response")
	}

	if resp.StatusCode > 299 {
		return gqlResp, fmt.Errorf("non-200 HTTP response code; %s", string(respJSON))
	}

	if gqlResp.Errors != nil && len(gqlResp.Errors) > 0 {
		return gqlResp, fmt.Errorf("graphQL error; path: %s, message: %s", gqlResp.Errors[0].Path, gqlResp.Errors[0].Message)
	}

	return gqlResp, nil
}
