package types

// CreateEnvironmentTokenRequest is a request to create an environment token.
type CreateEnvironmentTokenRequest struct {
	Verifier *RequestVerifier
	Env      string `json:"env"`
}

// CreateEnvironmentTokenResponse is a response to a create token request.
type CreateEnvironmentTokenResponse struct {
	Token string `json:"token"`
}
