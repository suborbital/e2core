package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/suborbital/appspec/system"
	"github.com/suborbital/e2core/common"
	"github.com/suborbital/e2core/options"
	"github.com/suborbital/vektor/vk"
)

const (
	AuthorizationCtxKey = "authorization"
	AccessExecute       = 8
)

func AuthorizationMiddleware(opts *options.Options, inner vk.HandlerFunc) vk.HandlerFunc {
	authorizer := NewApiAuthClient(opts)
	return func(req *http.Request, ctx *vk.Ctx) (interface{}, error) {
		identifier := ctx.Params.ByName("ident")
		namespace := ctx.Params.ByName("namespace")
		name := ctx.Params.ByName("name")

		authz, err := authorizer.Authorize(ExtractAccessToken(req.Header), identifier, namespace, name)
		if err != nil {
			ctx.Log.Error(err)
			return vk.R(http.StatusUnauthorized, ""), nil
		}

		ctx.Context = context.WithValue(ctx.Context, AuthorizationCtxKey, authz)

		return inner(req, ctx)
	}
}

func NewApiAuthClient(opts *options.Options) *AuthzClient {
	return &AuthzClient{
		httpClient: &http.Client{
			Timeout:   20 * time.Second,
			Transport: http.DefaultTransport,
		},
		location: opts.ControlPlane + "/api/v2/access",
		cache:    NewAuthorizationCache(opts.AuthCacheTTL),
	}
}

type AuthzClient struct {
	location   string
	httpClient *http.Client
	cache      *AuthorizationCache
}

func (client *AuthzClient) Authorize(token system.Credential, identifier, namespace, name string) (*AuthorizationContext, error) {
	if token == nil {
		return nil, common.Error(common.ErrAccess, "no credentials provided")
	}

	environment, tenant := SplitIdentifier(identifier)
	if environment == "" {
		return nil, common.Error(common.ErrAccess, "invalid identifier")
	}

	accessReq := &AuthorizationRequest{
		Action:   AccessExecute,
		Resource: []string{environment, tenant, namespace, name},
	}

	key := filepath.Join(identifier, namespace, name, token.Value())

	return client.cache.Get(key, client.loadAuth(token, accessReq))
}

func (client *AuthzClient) loadAuth(token system.Credential, req *AuthorizationRequest) func() (*AuthorizationContext, error) {
	return func() (*AuthorizationContext, error) {
		buf := bytes.NewBuffer([]byte{})
		if err := json.NewEncoder(buf).Encode(req); err != nil {
			return nil, common.Error(err, "serialized authorization request")
		}

		authzReq, err := http.NewRequest(http.MethodPost, client.location, buf)
		if err != nil {
			return nil, common.Error(err, "post authorization request")
		}

		// pass token along
		headerVal := fmt.Sprintf("%s %s", token.Scheme(), token.Value())
		authzReq.Header.Set(http.CanonicalHeaderKey("Authorization"), headerVal)

		resp, err := client.httpClient.Do(authzReq)
		if err != nil {
			return nil, common.Error(err, "dispatch remote authz request")
		}

		if resp.StatusCode != http.StatusOK {
			return nil, common.Error(common.ErrAccess, "non-200 response %d for authorization service", resp.StatusCode)
		}
		defer resp.Body.Close()

		var authz *AuthorizationResponse
		if err = json.NewDecoder(resp.Body).Decode(&authz); err != nil {
			return nil, common.Error(err, "deserialized authorization response")
		}

		return NewAuthorizationContext(authz), nil
	}
}

func NewAccessToken(token string) system.Credential {
	if token != "" {
		return &AccessToken{
			scheme: "Bearer",
			value:  token,
		}
	}

	return nil
}

var _ system.Credential = (*AccessToken)(nil)

type AccessToken struct {
	scheme string
	value  string
}

func (a AccessToken) Scheme() string {
	return a.scheme
}

func (a AccessToken) Value() string {
	return a.value
}

func (a AccessToken) String() string {
	return fmt.Sprintf("%s %s", a.scheme, a.value)
}

type (
	AuthorizationRequest struct {
		Action   uint8    `json:"action"`
		Resource []string `json:"resource"`
	}

	AuthorizationResponse struct {
		Identity    string `json:"identity"`
		Account     string `json:"account"`
		Environment string `json:"environment"`
		Tenant      string `json:"tenant"`
		Path        string `json:"path"`
	}
)

func NewAuthorizationContext(response *AuthorizationResponse) *AuthorizationContext {
	segments := strings.Split(response.Path, "/")
	return &AuthorizationContext{
		account:     response.Account,
		environment: response.Environment,
		tenant:      response.Tenant,
		namespace:   segments[0],
		module:      segments[1],
	}
}

type AuthorizationContext struct {
	account     string
	environment string
	tenant      string
	namespace   string
	module      string
}

func (authz *AuthorizationContext) Identity() string {
	return authz.account
}

func (authz *AuthorizationContext) Account() string {
	return authz.account
}

func (authz *AuthorizationContext) Environment() string {
	return authz.environment
}

func (authz *AuthorizationContext) Tenant() string {
	return authz.tenant
}

func (authz *AuthorizationContext) Namespace() string {
	return authz.namespace
}

func (authz *AuthorizationContext) Module() string {
	return authz.module
}

func ExtractAccessToken(header http.Header) system.Credential {
	authInfo := header.Get(http.CanonicalHeaderKey("Authorization"))
	if authInfo == "" {
		return nil
	}

	splitAt := strings.Index(authInfo, " ")
	if splitAt == 0 {
		return nil
	}

	return &AccessToken{
		scheme: authInfo[:splitAt],
		value:  authInfo[splitAt+1:],
	}
}
