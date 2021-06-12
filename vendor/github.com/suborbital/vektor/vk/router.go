package vk

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/julienschmidt/httprouter"
	"github.com/suborbital/vektor/vlog"
)

const contentTypeHeaderKey = "Content-Type"

// used internally to convey content types
type contentType string

// HandlerFunc is the vk version of http.HandlerFunc
// instead of exposing the ResponseWriter, the function instead returns
// an object and an error, which are handled as described in `With` below
type HandlerFunc func(*http.Request, *Ctx) (interface{}, error)

// Router handles the responses on behalf of the server
type Router struct {
	*RouteGroup                     // the "root" RouteGroup that is mounted at server start
	hrouter      *httprouter.Router // the internal 'actual' router
	finalizeOnce sync.Once          // ensure that the root only gets mounted once

	log *vlog.Logger
}

type defaultScope struct {
	RequestID string `json:"request_id"`
}

// NewRouter creates a new Router
func NewRouter(logger *vlog.Logger) *Router {
	// add the logger middleware
	middleware := []Middleware{loggerMiddleware()}

	r := &Router{
		RouteGroup:   Group("").Before(middleware...),
		hrouter:      httprouter.New(),
		finalizeOnce: sync.Once{},
		log:          logger,
	}

	return r
}

// HandleHTTP handles a classic Go HTTP handlerFunc
func (rt *Router) HandleHTTP(method, path string, handler http.HandlerFunc) {
	rt.hrouter.Handle(method, path, func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		handler(w, r)
	})
}

// Finalize mounts the root group to prepare the Router to handle requests
func (rt *Router) Finalize() {
	rt.finalizeOnce.Do(func() {
		rt.mountGroup(rt.RouteGroup)
	})
}

//ServeHTTP serves HTTP requests
func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// check to see if the router has a handler for this path
	handler, params, _ := rt.hrouter.Lookup(r.Method, r.URL.Path)

	if handler != nil {
		handler(w, r, params)
	} else {
		rt.log.Debug("not handled:", r.Method, r.URL.String())

		// let httprouter handle the fallthrough cases
		rt.hrouter.ServeHTTP(w, r)
	}
}

// mountGroup adds a group of handlers to the httprouter
func (rt *Router) mountGroup(group *RouteGroup) {
	for _, r := range group.routeHandlers() {
		rt.log.Debug("mounting route", r.Method, r.Path)
		rt.hrouter.Handle(r.Method, r.Path, rt.handleWrap(r.Handler))
	}
}

// handleWrap returns an httprouter.Handle that uses the `inner` vk.HandleFunc to handle the request
//
// inner returns a body and an error;
// the body can can be:
// - a vk.Response object (status and body are written to w)
// - []byte (written directly to w, status 200)
// - a struct (marshalled to JSON and written to w, status 200)
//
// the error can be:
// - a vk.Error type (status and message are written to w)
// - any other error object (status 500 and error.Error() are written to w)
//
func (rt *Router) handleWrap(inner HandlerFunc) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		var status int
		var body []byte
		var detectedCType contentType

		// create a context handleWrap the configured logger
		// (and use the ctx.Log for all remaining logging
		// in case a scope was set on it)
		ctx := NewCtx(rt.log, params, w.Header())
		ctx.UseScope(defaultScope{ctx.RequestID()})

		resp, err := inner(r, ctx)
		if err != nil {
			status, body, detectedCType = errorOrOtherToBytes(ctx.Log, err)
		} else {
			status, body, detectedCType = responseOrOtherToBytes(ctx.Log, resp)
		}

		// check if anything in the handler chain set the content type
		// header, and only use the auto-detected value if it wasn't
		headerCType := w.Header().Get(contentTypeHeaderKey)
		shouldSetCType := headerCType == ""

		ctx.Log.Debug("post-handler contenttype:", string(headerCType))

		// if no contentType was set in the middleware chain,
		// then set it here based on the type detected
		if shouldSetCType {
			ctx.Log.Debug("setting auto-detected contenttype:", string(detectedCType))
			w.Header().Set(contentTypeHeaderKey, string(detectedCType))
		}

		w.WriteHeader(status)
		w.Write(body)

		ctx.Log.Info(r.Method, r.URL.String(), fmt.Sprintf("completed (%d: %s)", status, http.StatusText(status)))
	}
}

// canHandle returns true if there's a registered handler that can
// handle the method and path provided or not
func (rt *Router) canHandle(method, path string) bool {
	handler, _, _ := rt.hrouter.Lookup(method, path)
	return handler != nil
}
