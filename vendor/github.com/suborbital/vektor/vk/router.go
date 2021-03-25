package vk

import (
	"fmt"
	"net/http"

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
	hrouter   *httprouter.Router
	root      *RouteGroup
	getLogger func() *vlog.Logger
}

type defaultScope struct {
	RequestID string `json:"request_id"`
}

// routerWithOptions returns a router with the specified options and optional middleware on the root route group
func routerWithOptions(options *Options, middleware ...Middleware) *Router {
	// add the logger middleware first
	middleware = append([]Middleware{loggerMiddleware()}, middleware...)

	r := &Router{
		hrouter: httprouter.New(),
		root:    Group("").Before(middleware...),
		getLogger: func() *vlog.Logger {
			return options.Logger
		},
	}

	return r
}

//ServeHTTP serves HTTP requests
func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// check to see if the router has a handler for this path
	handler, params, _ := rt.hrouter.Lookup(r.Method, r.URL.Path)

	if handler != nil {
		handler(w, r, params)
	} else {
		rt.getLogger().Debug("not handled:", r.Method, r.URL.String())

		// let httprouter handle the fallthrough cases
		rt.hrouter.ServeHTTP(w, r)
	}
}

// GET is a shortcut for router.Handle(http.MethodGet, path, handle)
func (rt *Router) GET(path string, handler HandlerFunc) {
	rt.root.GET(path, handler)
}

// HEAD is a shortcut for router.Handle(http.MethodHead, path, handle)
func (rt *Router) HEAD(path string, handler HandlerFunc) {
	rt.root.HEAD(path, handler)
}

// OPTIONS is a shortcut for router.Handle(http.MethodOptions, path, handle)
func (rt *Router) OPTIONS(path string, handler HandlerFunc) {
	rt.root.OPTIONS(path, handler)
}

// POST is a shortcut for router.Handle(http.MethodPost, path, handle)
func (rt *Router) POST(path string, handler HandlerFunc) {
	rt.root.POST(path, handler)
}

// PUT is a shortcut for router.Handle(http.MethodPut, path, handle)
func (rt *Router) PUT(path string, handler HandlerFunc) {
	rt.root.PUT(path, handler)
}

// PATCH is a shortcut for router.Handle(http.MethodPatch, path, handle)
func (rt *Router) PATCH(path string, handler HandlerFunc) {
	rt.root.PATCH(path, handler)
}

// DELETE is a shortcut for router.Handle(http.MethodDelete, path, handle)
func (rt *Router) DELETE(path string, handler HandlerFunc) {
	rt.root.DELETE(path, handler)
}

// Handle adds a route to be handled
func (rt *Router) Handle(method, path string, handler HandlerFunc) {
	rt.root.Handle(method, path, handler)
}

// HandleHTTP allows vk to handle a standard http.HandlerFunc
func (rt *Router) HandleHTTP(method, path string, handler http.HandlerFunc) {
	rt.hrouter.Handle(method, path, func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		handler(w, r)
	})
}

// AddGroup adds a group to the router's root group,
// which is mounted to the server upon Start()
func (rt *Router) AddGroup(group *RouteGroup) {
	rt.root.AddGroup(group)
}

// mountGroup adds a group of handlers to the httprouter
func (rt *Router) mountGroup(group *RouteGroup) {
	for _, r := range group.routeHandlers() {
		rt.getLogger().Debug("mounting route", r.Method, r.Path)
		rt.hrouter.Handle(r.Method, r.Path, rt.with(r.Handler))
	}
}

// rootGroup returns the root RouteGroup to be mounted before server start
func (rt *Router) rootGroup() *RouteGroup {
	return rt.root
}

// with returns an httprouter.Handle that uses the `inner` vk.HandleFunc to handle the request
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
func (rt *Router) with(inner HandlerFunc) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		var status int
		var body []byte
		var detectedCType contentType

		// create a context with the configured logger
		// (and use the ctx.Log for all remaining logging
		// in case a scope was set on it)
		ctx := NewCtx(rt.getLogger(), params, w.Header())
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
