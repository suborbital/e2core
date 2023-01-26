package transformers

import (
	"bytes"
	"io"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/suborbital/systemspec/request"
)

// FromEchoContext creates a CoordinatedRequest from an echo context.
func FromEchoContext(c echo.Context) (*request.CoordinatedRequest, error) {
	var err error
	reqBody := make([]byte, 0)
	if c.Request().Body != nil { // Read
		reqBody, err = io.ReadAll(c.Request().Body)
		if err != nil {
			return nil, errors.Wrap(err, "io.ReadAll request body")
		}
	}
	c.Request().Body = io.NopCloser(bytes.NewBuffer(reqBody)) // Reset

	flatHeaders := map[string]string{}
	for k, v := range c.Request().Header {
		// we lowercase the key to have case-insensitive lookup later
		flatHeaders[strings.ToLower(k)] = v[0]
	}

	flatParams := map[string]string{}
	for _, p := range c.ParamNames() {
		flatParams[p] = c.Param(p)
	}

	return &request.CoordinatedRequest{
		Method:      c.Request().Method,
		URL:         c.Request().URL.RequestURI(),
		ID:          c.Request().Header.Get("requestID"),
		Body:        reqBody,
		Headers:     flatHeaders,
		RespHeaders: map[string]string{},
		Params:      flatParams,
		State:       map[string][]byte{},
	}, nil
}

// UseSuborbitalHeaders adds the values in the state and params headers JSON to the CoordinatedRequest's State and Params
func UseSuborbitalHeaders(ec echo.Context, c *request.CoordinatedRequest) error {
	// fill in initial state from the state header
	stateJSON := ec.Request().Header.Get(suborbitalStateHeader)
	if err := c.addState(stateJSON); err != nil {
		return err
	}

	// fill in the URL params from the Params header
	paramsJSON := ec.Request().Header.Get(suborbitalParamsHeader)
	if err := c.addParams(paramsJSON); err != nil {
		return err
	}

	ec.Response().Header()[suborbitalRequestIDHeader] = []string{ec.Request().Header.Get("requestID")}

	return nil
}
