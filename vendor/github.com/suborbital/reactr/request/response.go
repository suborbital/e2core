package request

import "encoding/json"

type CoordinatedResponse struct {
	Output      []byte            `json:"output"`
	RespHeaders map[string]string `json:"resp_headers"`
}

// ToJSON returns a JSON representation of a CoordinatedRequest
func (c *CoordinatedResponse) ToJSON() ([]byte, error) {
	return json.Marshal(c)
}
