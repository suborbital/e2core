package scn

import (
	"net/http"

	"github.com/pkg/errors"

	"github.com/suborbital/e2core/e2/scn/types"
)

// SendHeartbeat sends a telemetry heartbeat request.
func (e *EnvironmentAPI) SendHeartbeat(heartbeat *types.HeartbeatRequest) error {
	uri := "/telemetry/v1/heartbeat"

	headers := map[string]string{
		tokenRequestHeaderKey: e.token,
	}

	if err := e.api.doWithHeaders(http.MethodPost, uri, headers, heartbeat, nil); err != nil {
		return errors.Wrap(err, "failed to doWithHeaders")
	}

	return nil
}
