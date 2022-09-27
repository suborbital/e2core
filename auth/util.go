package auth

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/pkg/errors"

	"github.com/suborbital/appspec/system"
)

// AuthorizationHeader communicates HTTP Challenge/Response data
type AuthorizationHeader struct {
	Scheme     string
	AuthParams []string
}

// ParseAuthorizationHeader reads the Authorization header value into an AuthorizationHeader instance.
func ParseAuthorizationHeader(header http.Header) (*AuthorizationHeader, error) {
	return ParseHttpAuthorization(header.Get("Authorization"))
}

// ParseHttpAuthorization reads headerValue into an AuthorizationHeader instance.
func ParseHttpAuthorization(headerValue string) (*AuthorizationHeader, error) {
	tokens := strings.SplitN(headerValue, " ", 2)
	if len(tokens) != 2 {
		return nil, errors.New("ParseHttpAuthorization: malformed authorization header")
	}

	return &AuthorizationHeader{
		Scheme:     tokens[0],
		AuthParams: strings.Split(tokens[1], ","),
	}, nil
}

// ParseBearerAuthHeader reads the Authorization header into a HttpBearerCredential
func ParseBearerAuthHeader(header http.Header) (*HttpBearerCredential, error) {
	authHeader, err := ParseAuthorizationHeader(header)
	if err != nil {
		return nil, err
	}

	if strings.ToLower(authHeader.Scheme) != "bearer" {
		return nil, errors.New("NewBearerTokenCredential: Auth Scheme must be Bearer")
	}

	if len(authHeader.AuthParams) < 1 {
		return nil, errors.New("NewBearerTokenCredential: Missing required credential")
	}

	return &HttpBearerCredential{
		token: authHeader.AuthParams[0],
	}, nil
}

// NewHttpBearerCredential returns a new instance of HttpBearerCredential with the provided userinfo.
func NewHttpBearerCredential(token string) *HttpBearerCredential {
	return &HttpBearerCredential{
		token: token,
	}
}

// HttpBearerCredential stores oauth bearer credentials used to perform HTTP Bearer authentication.
// See also https://datatracker.ietf.org/doc/html/rfc6750
type HttpBearerCredential struct {
	token string
}

// Scheme returns HTTP Auth Scheme.
func (credential *HttpBearerCredential) Scheme() string {
	return "Bearer"
}

// Value returns the users credentials in their serialized form.
func (credential *HttpBearerCredential) Value() string {
	return credential.token
}

// HeaderValue returns HttpBasicCredential in its HTTP Authorization Header format.
func (credential *HttpBearerCredential) HeaderValue() string {
	return fmt.Sprintf("%s %s", credential.Scheme(), credential.Value())
}

// AuthorizationHeaderValue returns a system.Credential in the http authorization header format.
func AuthorizationHeaderValue(credential system.Credential) string {
	return fmt.Sprintf("%s %s", credential.Scheme(), credential.Value())
}

func SplitIdentifier(identifier string) (string, string) {
	splitAt := strings.LastIndex(identifier, ".")
	if splitAt == -1 {
		return "", ""
	}

	return identifier[:splitAt], identifier[splitAt+1:]
}
