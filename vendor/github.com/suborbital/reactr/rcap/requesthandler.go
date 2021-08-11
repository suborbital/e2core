package rcap

import (
	"github.com/pkg/errors"
	"github.com/suborbital/reactr/request"
)

const (
	RequestFieldTypeMeta   = int32(0)
	RequestFieldTypeBody   = int32(1)
	RequestFieldTypeHeader = int32(2)
	RequestFieldTypeParams = int32(3)
	RequestFieldTypeState  = int32(4)
)

var (
	ErrReqNotSet        = errors.New("req is not set")
	ErrInvalidFieldType = errors.New("invalid field type")
	ErrInvalidKey       = errors.New("invalid key")
)

// RequestHandlerConfig is configuration for the request capability
type RequestHandlerConfig struct {
	Enabled bool `json:"enabled" yaml:"enabled"`
}

// RequestHandlerCapability allows runnables to handle HTTP requests
type RequestHandlerCapability interface {
	GetField(fieldType int32, key string) ([]byte, error)
	SetResponseHeader(key, val string) error
}

type requestHandler struct {
	config RequestHandlerConfig
	req    *request.CoordinatedRequest
}

// NewRequestHandler provides a handler for the given request
func NewRequestHandler(config RequestHandlerConfig, req *request.CoordinatedRequest) RequestHandlerCapability {
	d := &requestHandler{
		config: config,
		req:    req,
	}

	return d
}

func (r *requestHandler) GetField(fieldType int32, key string) ([]byte, error) {
	if !r.config.Enabled {
		return nil, ErrCapabilityNotEnabled
	}

	if r.req == nil {
		return nil, ErrReqNotSet
	}

	val := ""

	switch fieldType {
	case RequestFieldTypeMeta:
		switch key {
		case "method":
			val = r.req.Method
		case "url":
			val = r.req.URL
		case "id":
			val = r.req.ID
		case "body":
			val = string(r.req.Body)
		default:
			return nil, ErrInvalidKey
		}
	case RequestFieldTypeBody:
		bodyVal, err := r.req.BodyField(key)
		if err == nil {
			val = bodyVal
		} else {
			return nil, errors.Wrap(err, "failed to get BodyField")
		}
	case RequestFieldTypeHeader:
		header, ok := r.req.Headers[key]
		if ok {
			val = header
		} else {
			return nil, ErrInvalidKey
		}
	case RequestFieldTypeParams:
		param, ok := r.req.Params[key]
		if ok {
			val = param
		} else {
			return nil, ErrInvalidKey
		}
	case RequestFieldTypeState:
		stateVal, ok := r.req.State[key]
		if ok {
			val = string(stateVal)
		} else {
			return nil, ErrInvalidKey
		}
	default:
		return nil, ErrInvalidFieldType
	}

	return []byte(val), nil
}

// SetResponseHeader sets a header on the response
func (r *requestHandler) SetResponseHeader(key, val string) error {
	if !r.config.Enabled {
		return ErrCapabilityNotEnabled
	}

	if r.req == nil {
		return ErrReqNotSet
	}

	if r.req.RespHeaders == nil {
		r.req.RespHeaders = map[string]string{}
	}

	r.req.RespHeaders[key] = val

	return nil
}
