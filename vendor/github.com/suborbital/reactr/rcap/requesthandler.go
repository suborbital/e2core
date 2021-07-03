package rcap

import (
	"github.com/pkg/errors"
	"github.com/suborbital/reactr/request"
)

const (
	fieldTypeMeta   = int32(0)
	fieldTypeBody   = int32(1)
	fieldTypeHeader = int32(2)
	fieldTypeParams = int32(3)
	fieldTypeState  = int32(4)
)

var (
	ErrReqNotSet        = errors.New("req is not set")
	ErrInvalidFieldType = errors.New("invalid field type")
	ErrInvalidKey       = errors.New("invalid key")
)

type RequestHandler interface {
	SetReq(req *request.CoordinatedRequest)
	GetField(fieldType int32, key string) ([]byte, error)
	SetResponseHeader(key, val string) error
}

// defaultRequestHandler provides information about the request bound to a runnable
type defaultRequestHandler struct {
	req *request.CoordinatedRequest
}

// DefaultRequestHandler provides the default request info handler
func DefaultRequestHandler() RequestHandler {
	d := &defaultRequestHandler{}

	return d
}

// SetReq sets the request to be handled
func (d *defaultRequestHandler) SetReq(req *request.CoordinatedRequest) {
	d.req = req
}

func (d *defaultRequestHandler) GetField(fieldType int32, key string) ([]byte, error) {
	if d.req == nil {
		return nil, ErrReqNotSet
	}

	val := ""

	switch fieldType {
	case fieldTypeMeta:
		switch key {
		case "method":
			val = d.req.Method
		case "url":
			val = d.req.URL
		case "id":
			val = d.req.ID
		case "body":
			val = string(d.req.Body)
		default:
			return nil, ErrInvalidKey
		}
	case fieldTypeBody:
		bodyVal, err := d.req.BodyField(key)
		if err == nil {
			val = bodyVal
		} else {
			return nil, errors.Wrap(err, "failed to get BodyField")
		}
	case fieldTypeHeader:
		header, ok := d.req.Headers[key]
		if ok {
			val = header
		} else {
			return nil, ErrInvalidKey
		}
	case fieldTypeParams:
		param, ok := d.req.Params[key]
		if ok {
			val = param
		} else {
			return nil, ErrInvalidKey
		}
	case fieldTypeState:
		stateVal, ok := d.req.State[key]
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
func (d *defaultRequestHandler) SetResponseHeader(key, val string) error {
	if d.req == nil {
		return ErrReqNotSet
	}

	if d.req.RespHeaders == nil {
		d.req.RespHeaders = map[string]string{}
	}

	d.req.RespHeaders[key] = val

	return nil
}
