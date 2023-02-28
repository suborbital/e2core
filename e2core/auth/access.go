package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/suborbital/e2core/e2core/options"
	"github.com/suborbital/e2core/foundation/common"
	"github.com/suborbital/systemspec/system"
)

type TenantInfo struct {
	AuthorizedParty string `json:"authorized_party"`
	Environment     string `json:"environment"`
	ID              string `json:"id"`
	Name            string `json:"name"`
}

func AuthorizationMiddleware(opts *options.Options) echo.MiddlewareFunc {
	authorizer := NewApiAuthClient(opts)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			identifier := c.Param("ident")
			namespace := c.Param("namespace")
			name := c.Param("name")

			tntInfo, err := authorizer.Authorize(ExtractAccessToken(c.Request().Header), identifier, namespace, name)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized).SetInternal(err)
			}

			c.Set("ident", tntInfo.ID)

			return next(c)
		}
	}
}

func NewApiAuthClient(opts *options.Options) *AuthzClient {
	return &AuthzClient{
		httpClient: &http.Client{
			Timeout:   20 * time.Second,
			Transport: http.DefaultTransport,
		},
		location: opts.ControlPlane + "/environment/v1/tenant/",
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
		authzReq, err := http.NewRequest(http.MethodGet, client.location+identifier, nil)
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
