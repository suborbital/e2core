package coordinator

import (
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/pkg/errors"
	"github.com/suborbital/grav/grav"
	"github.com/suborbital/hive-wasm/directive"
	"github.com/suborbital/hive-wasm/wasm"
	"github.com/suborbital/hive/hive"
	"github.com/suborbital/vektor/vk"
)

// Coordinator is a type that is responsible for covnerting the directive into
// usable Vektor handles by coordinating Hive jobs and meshing when needed.
type Coordinator struct {
	directive *directive.Directive
	bundle    *wasm.Bundle

	hive *hive.Hive
	bus  *grav.Grav

	lock sync.RWMutex
}

// New creates a coordinator
func New() *Coordinator {
	hive := hive.New()
	grav := grav.New()

	c := &Coordinator{
		hive: hive,
		bus:  grav,
		lock: sync.RWMutex{},
	}

	return c
}

// UseBundle sets a bundle to be used
func (c *Coordinator) UseBundle(bundle *wasm.Bundle) *vk.RouteGroup {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.directive = bundle.Directive

	wasm.HandleBundle(c.hive, bundle)

	group := vk.Group("")

	for _, h := range bundle.Directive.Handlers {
		if h.Input.Type != directive.DirectiveInputTypeRequest {
			continue
		}

		handler := c.vkHandlerForDirectiveHandler(h)

		group.Handle(h.Input.Method, h.Input.Resource, handler)
	}

	return group
}

func (c *Coordinator) vkHandlerForDirectiveHandler(handler directive.Handler) vk.HandlerFunc {
	return func(r *http.Request, ctx *vk.Ctx) (interface{}, error) {
		reqBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return nil, vk.E(http.StatusInternalServerError, "failed to read request body")
		}

		defer r.Body.Close()

		resp := []interface{}{}

		for _, step := range handler.Steps {
			// if the group is nil, call the single func
			if step.Group == nil || len(step.Group) > 0 {
				result, err := c.runSingleFn(step.Fn, reqBody)
				if err != nil {
					return nil, err
				}

				entry := map[string]interface{}{
					step.Fn: result,
				}

				resp = append(resp, entry)
			} else {
				return nil, vk.E(http.StatusInternalServerError, "cannot handle function groups yet")
			}
		}

		return resp, nil
	}
}

func (c *Coordinator) runSingleFn(name string, body []byte) (interface{}, error) {
	job := hive.NewJob(name, body)

	result, err := c.hive.Do(job).Then()
	if err != nil {
		if vkErr, isVkErr := err.(vk.Error); isVkErr {
			return nil, vkErr
		}

		return nil, vk.Wrap(http.StatusInternalServerError, errors.Wrapf(err, "fn %s failed", name))
	}

	return string(result.([]byte)), nil
}
