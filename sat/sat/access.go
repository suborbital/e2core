package sat

import "github.com/suborbital/appspec/system"

var _ system.Credential = (*AuthToken)(nil)

type AuthToken struct {
	scheme string
	value  string
}

func (a AuthToken) Scheme() string {
	return a.scheme
}

func (a AuthToken) Value() string {
	return a.value
}

func NewAuthToken(token string) system.Credential {
	if token != "" {
		return &AuthToken{
			scheme: "Bearer",
			value:  token,
		}
	}

	return nil
}
