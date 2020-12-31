package vk

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	"github.com/suborbital/vektor/vlog"
)

// ctxKey is a type to represent a key in the Ctx context.
type ctxKey string

// Ctx serves a similar purpose to context.Context, but has some typed fields
type Ctx struct {
	Context     context.Context
	Log         *vlog.Logger
	Params      httprouter.Params
	RespHeaders http.Header
	requestID   string
	scope       interface{}
}

// NewCtx creates a new Ctx
func NewCtx(log *vlog.Logger, params httprouter.Params, headers http.Header) *Ctx {
	ctx := &Ctx{
		Context:     context.Background(),
		Log:         log,
		Params:      params,
		RespHeaders: headers,
	}

	return ctx
}

// Set sets a value on the Ctx's embedded Context (a la key/value store)
func (c *Ctx) Set(key string, val interface{}) {
	realKey := ctxKey(key)
	c.Context = context.WithValue(c.Context, realKey, val)
}

// Get gets a value from the Ctx's embedded Context (a la key/value store)
func (c *Ctx) Get(key string) interface{} {
	realKey := ctxKey(key)
	val := c.Context.Value(realKey)

	return val
}

// UseScope sets an object to be the scope of the request, including setting the logger's scope
// the scope can be retrieved later with the Scope() method
func (c *Ctx) UseScope(scope interface{}) {
	c.Log = c.Log.CreateScoped(scope)

	c.scope = scope
}

// Scope retrieves the context's scope
func (c *Ctx) Scope() interface{} {
	return c.scope
}

// UseRequestID is a setter for the request ID
func (c *Ctx) UseRequestID(id string) {
	c.requestID = id
}

// RequestID returns the request ID of the current request, generating one if none exists.
func (c *Ctx) RequestID() string {
	if c.requestID == "" {
		c.requestID = uuid.New().String()
	}

	return c.requestID
}
