package appsource

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/suborbital/atmo/atmo/options"
	"github.com/suborbital/atmo/directive"
	"github.com/suborbital/atmo/directive/executable"
	"github.com/suborbital/atmo/fqfn"
	"github.com/suborbital/reactr/rcap"
)

// HTTPSource is an AppSource backed by an HTTP client connected to a remote source
type HTTPSource struct {
	host string
	opts options.Options

	runnables map[string]directive.Runnable
	lock      sync.RWMutex
}

// NewHTTPSource creates a new HTTPSource that looks for a bundle at [host]
func NewHTTPSource(host string) AppSource {
	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		host = fmt.Sprintf("http://%s", host)
	}

	b := &HTTPSource{
		host:      host,
		runnables: map[string]directive.Runnable{},
		lock:      sync.RWMutex{},
	}

	return b
}

// Start initializes the app source
func (h *HTTPSource) Start(opts options.Options) error {
	h.opts = opts

	if err := h.pingServer(); err != nil {
		return errors.Wrap(err, "failed to findBundle")
	}

	return nil
}

// Runnables returns the Runnables for the app
func (h *HTTPSource) Runnables() []directive.Runnable {
	// if we're in headless mode, only return the Runnables we've cached
	// from calls to FindRunnable (we don't want to load EVERY Runnable)
	if *h.opts.Headless {
		h.lock.RLock()
		defer h.lock.RUnlock()

		return h.headlessRunnableList()
	}

	runnables := []directive.Runnable{}
	if _, err := h.get("/runnables", &runnables); err != nil {
		h.opts.Logger.Error(errors.Wrap(err, "failed to get /runnables"))
	}

	return runnables
}

// FindRunnable returns a nil error if a Runnable with the
// provided FQFN can be made available at the next sync,
// otherwise ErrRunnableNotFound is returned
func (h *HTTPSource) FindRunnable(FQFN, auth string) (*directive.Runnable, error) {
	parsedFQFN := fqfn.Parse(FQFN)

	// if we are in headless mode, first check to see if we've cached a runnable
	if *h.opts.Headless {
		h.lock.RLock()
		r, ok := h.runnables[FQFN]
		h.lock.RUnlock()

		if ok {
			return &r, nil
		}
	}

	// if we need to fetch it from remote, let's do so

	path := fmt.Sprintf("/runnable%s", parsedFQFN.HeadlessURLPath())

	runnable := directive.Runnable{}
	if resp, err := h.authedGet(path, auth, &runnable); err != nil {
		h.opts.Logger.Error(errors.Wrapf(err, "failed to get %s", path))

		if resp.StatusCode == http.StatusUnauthorized {
			return nil, ErrAuthenticationFailed
		}

		return nil, ErrRunnableNotFound
	}

	if auth != "" {
		// if we get this far, we assume the token was used to successfully get
		// the runnable from the control plane, and should therefore be used to
		// authenticate further calls for this function, so we cache it
		runnable.TokenHash = TokenHash(auth)
	}

	// again, if we're in headless mode let's cache it for later
	if *h.opts.Headless {
		h.lock.Lock()
		defer h.lock.Unlock()

		h.runnables[runnable.FQFN] = runnable
	}

	return &runnable, nil
}

// Handlers returns the handlers for the app
func (h *HTTPSource) Handlers() []directive.Handler {
	if *h.opts.Headless {
		h.lock.RLock()
		defer h.lock.RUnlock()

		return h.headlessHandlers()
	}

	handlers := []directive.Handler{}
	if _, err := h.get("/handlers", &handlers); err != nil {
		h.opts.Logger.Error(errors.Wrap(err, "failed to get /handlers"))
	}

	return handlers
}

// Schedules returns the schedules for the app
func (h *HTTPSource) Schedules() []directive.Schedule {
	schedules := []directive.Schedule{}
	if _, err := h.get("/schedules", &schedules); err != nil {
		h.opts.Logger.Error(errors.Wrap(err, "failed to get /schedules"))
	}

	return schedules
}

// Connections returns the Connections for the app
func (h *HTTPSource) Connections() directive.Connections {
	connections := directive.Connections{}
	if _, err := h.get("/connections", &connections); err != nil {
		h.opts.Logger.Error(errors.Wrap(err, "failed to get /connections"))
	}

	return connections
}

// Authentication returns the Authentication for the app
func (h *HTTPSource) Authentication() directive.Authentication {
	authentication := directive.Authentication{}
	if _, err := h.get("/authentication", &authentication); err != nil {
		h.opts.Logger.Error(errors.Wrap(err, "failed to get /authentication"))
	}

	return authentication
}

// Capabilities returns the Capabilities for the app
func (h *HTTPSource) Capabilities() *rcap.CapabilityConfig {
	capabilities := rcap.CapabilityConfig{}
	if _, err := h.get("/capabilities", &capabilities); err != nil {
		h.opts.Logger.Error(errors.Wrap(err, "failed to get /authentication"))
	}

	return &capabilities
}

// File returns a requested file
func (h *HTTPSource) File(filename string) ([]byte, error) {
	path := fmt.Sprintf("/file/%s", filename)

	resp, err := h.get(path, nil)
	if err != nil {
		h.opts.Logger.Error(errors.Wrapf(err, "failed to get %s", path))
		return nil, os.ErrNotExist
	}

	defer resp.Body.Close()
	file, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to ReadAll")
	}

	return file, nil
}

// Queries returns the Queries for the app
func (h *HTTPSource) Queries() []directive.DBQuery {
	queries := []directive.DBQuery{}
	if _, err := h.get("/queries", &queries); err != nil {
		h.opts.Logger.Error(errors.Wrap(err, "failed to get /queries"))
	}

	return queries
}

func (h *HTTPSource) Meta() Meta {
	meta := Meta{}
	if _, err := h.get("/meta", &meta); err != nil {
		h.opts.Logger.Error(errors.Wrap(err, "failed to get /meta"))
	}

	return meta
}

// pingServer loops forever until it finds a server at the configured host
func (h *HTTPSource) pingServer() error {
	for {
		if _, err := h.get("/meta", nil); err != nil {

			if h.opts.Wait == nil || !*h.opts.Wait {
				return errors.Wrapf(err, "failed to connect to source at %s", h.host)
			}

			h.opts.Logger.Warn("failed to connect to remote source, will retry:", err.Error())

			time.Sleep(time.Second)

			continue
		}

		h.opts.Logger.Debug("connected to remote source at", h.host)

		break
	}

	return nil
}

// get performs a GET request against the configured host and given path
func (h *HTTPSource) get(path string, dest interface{}) (*http.Response, error) {
	return h.authedGet(path, "", dest)
}

// authedGet performs a GET request against the configured host and given path with the given auth header
func (h *HTTPSource) authedGet(path, auth string, dest interface{}) (*http.Response, error) {
	url, err := url.Parse(fmt.Sprintf("%s%s", h.host, path))
	if err != nil {
		return nil, errors.Wrap(err, "failed to url.Parse")
	}

	req, err := http.NewRequest(http.MethodGet, url.String(), nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to NewRequest")
	}

	if auth != "" {
		req.Header.Set("Authorization", auth)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to Do request")
	}

	if resp.StatusCode != http.StatusOK {
		return resp, fmt.Errorf("response returned non-200 status: %d", resp.StatusCode)
	}

	if dest != nil {
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, errors.Wrap(err, "failed to ReadAll body")
		}

		if err := json.Unmarshal(body, dest); err != nil {
			return nil, errors.Wrap(err, "failed to json.Unmarshal")
		}
	}

	return resp, nil
}

func (h *HTTPSource) headlessRunnableList() []directive.Runnable {
	runnables := []directive.Runnable{}

	for _, r := range h.runnables {
		runnables = append(runnables, r)
	}

	return runnables
}

func (h *HTTPSource) headlessHandlers() []directive.Handler {
	handlers := []directive.Handler{}

	// for each Runnable, construct a handler that executes it
	// based on a POST request to its FQFN URL /identifier/namespace/fn/version
	for _, runnable := range h.headlessRunnableList() {
		handler := directive.Handler{
			Input: directive.Input{
				Type:     directive.InputTypeRequest,
				Method:   http.MethodPost,
				Resource: fqfn.Parse(runnable.FQFN).HeadlessURLPath(),
			},
			Steps: []executable.Executable{
				{
					CallableFn: executable.CallableFn{
						Fn:   runnable.Name,
						With: map[string]string{},
						FQFN: runnable.FQFN,
					},
				},
			},
		}

		handlers = append(handlers, handler)
	}

	return handlers
}
