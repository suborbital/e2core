package scn

import (
	"net/http"

	"github.com/pkg/errors"

	"github.com/suborbital/e2core/e2/scn/types"
)

// CreateEnvironmentToken creates an environment token.
func (a *VerifiedAPI) CreateEnvironmentToken() (*types.CreateEnvironmentTokenResponse, error) {
	uri := "/auth/v1/token"

	req := &types.CreateEnvironmentTokenRequest{
		Verifier: a.verifier,
		Env:      "",
	}

	resp := &types.CreateEnvironmentTokenResponse{}
	if err := a.api.do(http.MethodPost, uri, req, resp); err != nil {
		return nil, errors.Wrap(err, "failed to Do")
	}

	return resp, nil
}
