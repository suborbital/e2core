package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/pkg/errors"

	"github.com/suborbital/appspec/system"
	"github.com/suborbital/vektor/vk"
)

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

const (
	AuthorizationCtxKey = "authorization"
	AccessExecute       = 8
)

func AuthorizationMiddleware(client *http.Client, controlplane string) vk.Middleware {
	authority := fmt.Sprintf("%s/%s", controlplane, "api/v2/access")
	return func(req *http.Request, ctx *vk.Ctx) error {
		identifier := ctx.Params.ByName("ident")
		namespace := ctx.Params.ByName("namespace")
		name := ctx.Params.ByName("name")

		environment, tenant := SplitIdentifier(identifier)
		if environment == "" {
			return vk.E(http.StatusBadRequest, "invalid identifier")
		}

		buf := bytes.NewBuffer([]byte{})

		accessReq := &AuthorizationRequest{
			Action:   AccessExecute,
			Resource: []string{environment, tenant, namespace, name},
		}

		if err := json.NewEncoder(buf).Encode(accessReq); err != nil {
			ctx.Log.Error(errors.Wrap(err, "serialize authorization request"))
			return vk.E(http.StatusUnauthorized, "")
		}

		authzReq, err := http.NewRequest(http.MethodPost, authority, buf)
		if err != nil {
			ctx.Log.Error(errors.Wrap(err, "create authorization request"))
			return vk.E(http.StatusUnauthorized, "")
		}
		// pass token along
		authzReq.Header.Set(http.CanonicalHeaderKey("Authorization"), req.Header.Get(http.CanonicalHeaderKey("Authorization")))

		resp, err := client.Do(authzReq)
		if err != nil {
			ctx.Log.Error(errors.Wrap(err, "post authorization request"))
			return vk.E(http.StatusUnauthorized, "")
		}

		if resp.StatusCode != http.StatusOK {
			ctx.Log.ErrorString(fmt.Sprintf("Request unauthorized %d", resp.StatusCode))
			return vk.E(http.StatusForbidden, "")
		}
		defer resp.Body.Close()

		var authz *AuthorizationResponse
		if err = json.NewDecoder(resp.Body).Decode(&authz); err != nil {
			ctx.Log.Error(errors.Wrap(err, "deserialized authorization response"))
			return vk.E(http.StatusInternalServerError, "")
		}

		ctx.Context = context.WithValue(ctx.Context, AuthorizationCtxKey, NewAuthorizationContext(authz))

		return nil
	}
}
