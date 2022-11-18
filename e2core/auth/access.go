package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/suborbital/e2core/e2core/options"
	"github.com/suborbital/e2core/foundation/common"
	"github.com/suborbital/systemspec/system"
	"github.com/suborbital/vektor/vk"
)

type TenantInfo struct {
	AuthorizedParty string `json:"authorized_party"`
	Organization    string `json:"organization"`
	Environment     string `json:"environment"`
	Tenant          string `json:"tenant"`
}

func AuthorizationMiddleware(opts *options.Options, inner vk.HandlerFunc) vk.HandlerFunc {
	authorizer := NewApiAuthClient(opts)
	return func(req *http.Request, ctx *vk.Ctx) (interface{}, error) {
		identifier := ctx.Params.ByName("ident")
		namespace := ctx.Params.ByName("namespace")
		name := ctx.Params.ByName("name")

		tntInfo, err := authorizer.Authorize(ExtractAccessToken(req.Header), identifier, namespace, name)
		if err != nil {
			ctx.Log.Error(err)
			return vk.R(http.StatusUnauthorized, ""), nil
		}

		ctx.Set("ident", fmt.Sprintf("%s.%s", tntInfo.Environment, tntInfo.Tenant))

		return inner(req, ctx)
	}
}

func NewApiAuthClient(opts *options.Options) *AuthzClient {
	return &AuthzClient{
		httpClient: &http.Client{
			Timeout:   20 * time.Second,
			Transport: http.DefaultTransport,
		},
		location: opts.ControlPlane + "/api/v2/tenant/%s",
		cache:    NewAuthorizationCache(opts.AuthCacheTTL),
	}
}

type AuthzClient struct {
	location   string
	httpClient *http.Client
	cache      *AuthorizationCache
}

func (client *AuthzClient) Authorize(token system.Credential, identifier, namespace, name string) (*TenantInfo, error) {
	if token == nil {
		return nil, common.Error(common.ErrAccess, "no credentials provided")
	}

	key := filepath.Join(identifier, namespace, name, token.Value())

	return client.cache.Get(key, client.loadAuth(token, identifier))
}

func (client *AuthzClient) loadAuth(token system.Credential, identifier string) func() (*TenantInfo, error) {
	return func() (*TenantInfo, error) {
		authzReq, err := http.NewRequest(http.MethodGet, fmt.Sprintf(client.location, identifier), nil)
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

		var claims *TenantInfo
		if err = json.NewDecoder(resp.Body).Decode(&claims); err != nil {
			return nil, common.Error(err, "deserialized authorization response")
		}

		return claims, nil
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
