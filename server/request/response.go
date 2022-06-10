package request

import (
	"encoding/json"

	"github.com/pkg/errors"
)

type CoordinatedResponse struct {
	Output      []byte            `json:"output"`
	RespHeaders map[string]string `json:"resp_headers"`
}

// ToJSON returns a JSON representation of a CoordinatedRequest
func (c *CoordinatedResponse) ToJSON() ([]byte, error) {
	return json.Marshal(c)
}

// ResponseFromJSON unmarshalls a CoordinatedResponse from JSON
func ResponseFromJSON(jsonBytes []byte) (*CoordinatedResponse, error) {
	resp := CoordinatedResponse{}
	if err := json.Unmarshal(jsonBytes, &resp); err != nil {
		return nil, errors.Wrap(err, "failed to Unmarshal respuest")
	}

	return &resp, nil
}
