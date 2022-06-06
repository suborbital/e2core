package capabilities

import (
	"net/url"
	"strings"

	"github.com/pkg/errors"
	"github.com/suborbital/velocity/server/request"
)

const (
	RequestFieldTypeMeta   = int32(0)
	RequestFieldTypeBody   = int32(1)
	RequestFieldTypeHeader = int32(2)
	RequestFieldTypeParams = int32(3)
	RequestFieldTypeState  = int32(4)
	RequestFieldTypeQuery  = int32(5)
)

var (
	ErrReqNotSet        = errors.New("req is not set")
	ErrInvalidFieldType = errors.New("invalid field type")
	ErrKeyNotFound      = errors.New("key not found")
)

// RequestHandlerConfig is configuration for the request capability
type RequestHandlerConfig struct {
	Enabled       bool `json:"enabled" yaml:"enabled"`
	AllowGetField bool `json:"allowGetField" yaml:"allowGetField"`
	AllowSetField bool `json:"allowSetField" yaml:"allowSetField"`
}

// RequestHandlerCapability allows runnables to handle HTTP requests
type RequestHandlerCapability interface {
	GetField(fieldType int32, key string) ([]byte, error)
	SetField(fieldType int32, key string, val string) error
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

// GetField gets a field from the attached request
func (r *requestHandler) GetField(fieldType int32, key string) ([]byte, error) {
	if !r.config.Enabled {
		return nil, ErrCapabilityNotEnabled
	} else if !r.config.AllowGetField {
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
			return nil, ErrKeyNotFound
		}
	case RequestFieldTypeBody:
		bodyVal, err := r.req.BodyField(key)
		if err == nil {
			val = bodyVal
		} else {
			return nil, errors.Wrap(err, "failed to get BodyField")
		}
	case RequestFieldTypeHeader:
		// lowercase to make the search case-insensitive
		lowerKey := strings.ToLower(key)
		header, ok := r.req.Headers[lowerKey]
		if ok {
			val = header
		} else {
			return nil, ErrKeyNotFound
		}
	case RequestFieldTypeParams:
		param, ok := r.req.Params[key]
		if ok {
			val = param
		} else {
			return nil, ErrKeyNotFound
		}
	case RequestFieldTypeState:
		stateVal, ok := r.req.State[key]
		if ok {
			val = string(stateVal)
		} else {
			return nil, ErrKeyNotFound
		}
	case RequestFieldTypeQuery:
		url, err := url.Parse(r.req.URL)
		if err != nil {
			return nil, errors.Wrap(err, "failed to url.Parse")
		}

		val = url.Query().Get(key)
	default:
		return nil, errors.Wrapf(ErrInvalidFieldType, "module requested field type %d", fieldType)
	}

	return []byte(val), nil
}

// SetField sets a field on the attached request
func (r *requestHandler) SetField(fieldType int32, key string, val string) error {
	if !r.config.Enabled {
		return ErrCapabilityNotEnabled
	} else if !r.config.AllowSetField {
		return ErrCapabilityNotEnabled
	}

	if r.req == nil {
		return ErrReqNotSet
	}

	switch fieldType {
	case RequestFieldTypeMeta:
		switch key {
		case "method":
			r.req.Method = val
		case "url":
			r.req.URL = val
		case "id":
			// do nothing
		case "body":
			r.req.Body = []byte(val)
		default:
			return ErrKeyNotFound
		}
	case RequestFieldTypeBody:
		if err := r.req.SetBodyField(key, val); err != nil {
			return errors.Wrap(err, "failed to get SetBodyField")
		}
	case RequestFieldTypeHeader:
		r.req.Headers[key] = val
	case RequestFieldTypeParams:
		r.req.Params[key] = val
	case RequestFieldTypeState:
		r.req.State[key] = []byte(val)
	default:
		return ErrInvalidFieldType
	}

	return nil
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
