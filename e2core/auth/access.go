package auth

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/suborbital/systemspec/system"
)

const (
	DefaultCacheTTL     = 10 * time.Minute
	DefaultCacheTTClean = 2 * time.Minute
)

type TenantInfo struct {
	AuthorizedParty string `json:"authorized_party"`
	Environment     string `json:"environment"`
	ID              string `json:"id"`
	Name            string `json:"name"`
}

type Authorizer interface {
	Authorize(token system.Credential, identifier, namespace, name string) (TenantInfo, error)
}

func AuthorizationMiddleware(authorizer Authorizer) echo.MiddlewareFunc {
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

// deriveKey is a utility function that takes a system.Credential token, identifier, namespace, and plugin name as args
// and returns one long string we can use as cache keys.
func deriveKey(token system.Credential, identifier, namespace, name string) (string, error) {
	if token == nil {
		return "", errors.New("token provided was nil")
	}

	return filepath.Join(identifier, namespace, name, token.Value()), nil
}
