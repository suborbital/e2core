package request

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
	"github.com/suborbital/vektor/vk"
)

// CoordinatedRequest represents a request whose fulfillment can be coordinated across multiple hosts
// and is serializable to facilitate interoperation with Wasm Runnables and transmissible over the wire
type CoordinatedRequest struct {
	Method      string            `json:"method"`
	URL         string            `json:"url"`
	ID          string            `json:"request_id"`
	Body        []byte            `json:"body"`
	Headers     map[string]string `json:"headers"`
	RespHeaders map[string]string `json:"resp_headers"`
	Params      map[string]string `json:"params"`
	State       map[string][]byte `json:"state"`

	bodyValues map[string]interface{} `json:"-"`
}

// FromVKRequest creates a CoordinatedRequest from an http.Request
func FromVKRequest(r *http.Request, ctx *vk.Ctx) (*CoordinatedRequest, error) {
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, vk.E(http.StatusInternalServerError, "failed to read request body")
	}

	defer r.Body.Close()

	flatHeaders := map[string]string{}
	for k, v := range r.Header {
		flatHeaders[k] = v[0]
	}

	flatParams := map[string]string{}
	for _, p := range ctx.Params {
		flatParams[p.Key] = p.Value
	}

	req := &CoordinatedRequest{
		Method:      r.Method,
		URL:         r.URL.RequestURI(),
		ID:          ctx.RequestID(),
		Body:        reqBody,
		Headers:     flatHeaders,
		RespHeaders: map[string]string{},
		Params:      flatParams,
		State:       map[string][]byte{},
	}

	return req, nil
}

// BodyField returns a field from the request body as a string
func (c *CoordinatedRequest) BodyField(key string) (string, error) {
	if c.bodyValues == nil {
		if len(c.Body) == 0 {
			return "", nil
		}

		vals := map[string]interface{}{}

		if err := json.Unmarshal(c.Body, &vals); err != nil {
			return "", errors.Wrap(err, "failed to Unmarshal request body")
		}

		// cache the parsed body
		c.bodyValues = vals
	}

	interfaceVal, ok := c.bodyValues[key]
	if !ok {
		return "", fmt.Errorf("body does not contain field %s", key)
	}

	stringVal, ok := interfaceVal.(string)
	if !ok {
		return "", fmt.Errorf("request body value %s is not a string", key)
	}

	return stringVal, nil
}

// FromJSON unmarshalls a CoordinatedRequest from JSON
func FromJSON(jsonBytes []byte) (*CoordinatedRequest, error) {
	req := CoordinatedRequest{}
	if err := json.Unmarshal(jsonBytes, &req); err != nil {
		return nil, errors.Wrap(err, "failed to Unmarshal request")
	}

	if req.Method == "" || req.URL == "" || req.ID == "" {
		return nil, errors.New("JSON is not CoordinatedRequest, required fields are empty")
	}

	return &req, nil
}

// ToJSON returns a JSON representation of a CoordinatedRequest
func (c *CoordinatedRequest) ToJSON() ([]byte, error) {
	return json.Marshal(c)
}
