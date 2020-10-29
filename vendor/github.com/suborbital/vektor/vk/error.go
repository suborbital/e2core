package vk

import (
	"encoding/json"
	"fmt"

	"github.com/suborbital/vektor/vlog"
)

// Error is an interface representing a failed request
type Error interface {
	Error() string // this ensures all Errors will also conform to the normal error interface

	Message() string
	Status() int
}

// ErrorResponse is a concrete implementation of Error,
// representing a failed HTTP request
type ErrorResponse struct {
	StatusCode  int    `json:"status"`
	MessageText string `json:"message"`
}

// Error returns a full error string
func (e *ErrorResponse) Error() string {
	return fmt.Sprintf("%d: %s", e.StatusCode, e.MessageText)
}

// Status returns the error status code
func (e *ErrorResponse) Status() int {
	return e.StatusCode
}

// Message returns the error's message
func (e *ErrorResponse) Message() string {
	return e.MessageText
}

// Err returns an error with status and message
func Err(status int, message string) Error {
	e := &ErrorResponse{
		StatusCode:  status,
		MessageText: message,
	}

	return e
}

// E is Err for those who like terse code
func E(status int, message string) Error {
	return Err(status, message)
}

// Wrap wraps an error in vk.Error
func Wrap(status int, err error) Error {
	return Err(status, err.Error())
}

var (
	genericErrorResponseBytes = []byte("Internal Server Error")
	genericErrorResponseCode  = 500
)

// converts _something_ into bytes, best it can:
// if data is Error type, returns (status, {status: status, message: message})
// if other error, returns (500, []byte(err.Error()))
func errorOrOtherToBytes(l *vlog.Logger, err error) (int, []byte, contentType) {
	statusCode := genericErrorResponseCode

	// first, check if it's vk.Error interface type, and unpack it for further processing
	if e, ok := err.(Error); ok {
		statusCode = e.Status() // grab this in case anything fails

		errResp := Err(e.Status(), e.Message()) // create a concrete instance that can be marshalled

		errJSON, marshalErr := json.Marshal(errResp)
		if marshalErr != nil {
			// any failure results in the generic response body being used
			l.ErrorString("failed to marshal vk.Error:", marshalErr.Error(), "original error:", err.Error())

			return statusCode, genericErrorResponseBytes, contentTypeTextPlain
		}

		return statusCode, errJSON, contentTypeJSON
	}

	l.Warn("redacting potential unsafe error response, original error:", err.Error())

	return statusCode, genericErrorResponseBytes, contentTypeTextPlain
}
