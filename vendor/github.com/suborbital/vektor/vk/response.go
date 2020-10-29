package vk

import (
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"
	"github.com/suborbital/vektor/vlog"
)

// Response represents a non-error HTTP response
type Response struct {
	status int
	body   interface{}
}

// Respond returns a filled-in response
func Respond(status int, body interface{}) Response {
	r := Response{
		status: status,
		body:   body,
	}

	return r
}

// R is `Respond` for those who prefer terse code
func R(status int, body interface{}) Response {
	return Respond(status, body)
}

// TODO: add convenience helpers for status codes

const (
	contentTypeJSON        contentType = "application/json"
	contentTypeTextPlain   contentType = "text/plain"
	contentTypeOctetStream contentType = "application/octet-stream"
)

// converts _something_ into bytes, best it can:
// if data is Response type, returns (status, body processed as below)
// if bytes, return (200, bytes)
// if string, return (200, []byte(string))
// if struct, return (200, json(struct))
// otherwise, return (500, nil)
func responseOrOtherToBytes(l *vlog.Logger, data interface{}) (int, []byte, contentType) {
	if data == nil {
		return http.StatusNoContent, []byte{}, contentTypeTextPlain
	}

	statusCode := http.StatusOK
	realData := data

	// first, check if it's response type, and unpack it for further processing
	if r, ok := data.(Response); ok {
		statusCode = r.status
		realData = r.body
	}

	// if data is []byte or string, return it as-is
	if b, ok := realData.([]byte); ok {
		return statusCode, b, contentTypeOctetStream
	} else if s, ok := realData.(string); ok {
		return statusCode, []byte(s), contentTypeTextPlain
	}

	// otherwise, assume it's a struct of some kind,
	// so JSON marshal it and return it
	json, err := json.Marshal(realData)
	if err != nil {
		l.Error(errors.Wrap(err, "failed to Marshal response struct"))

		return genericErrorResponseCode, []byte(genericErrorResponseBytes), contentTypeTextPlain
	}

	return statusCode, json, contentTypeJSON
}
