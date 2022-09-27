package auth

import "github.com/suborbital/appspec/system"

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

func NewAccessToken(token string) system.Credential {
	if token != "" {
		return &AccessToken{
			scheme: "Bearer",
			value:  token,
		}
	}

	return nil
}
