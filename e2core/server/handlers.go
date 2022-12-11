package server

import (
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	"github.com/suborbital/e2core/e2core/sequence"
	"github.com/suborbital/systemspec/request"
	"github.com/suborbital/systemspec/tenant"
	"github.com/suborbital/vektor/vk"
)

func (s *Server) executePluginByNameHandler() vk.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, ctx *vk.Ctx) error {
		ident := readParam(ctx, "ident")
		namespace := readParam(ctx, "namespace")
		name := readParam(ctx, "name")

		mod := s.syncer.GetModuleByName(ident, namespace, name)
		if mod == nil {
			return vk.E(http.StatusNotFound, "module not found")
		}

		req, err := request.FromVKRequest(r, ctx)
		if err != nil {
			ctx.Log.Error(errors.Wrap(err, "failed to request.FromVKRequest"))
			return vk.E(http.StatusInternalServerError, "failed to handle request")
		}

		req.UseSuborbitalHeaders(r, ctx)

		steps := []tenant.WorkflowStep{{FQMN: mod.FQMN}}

		// a sequence executes the handler's steps and manages its state.
		seq, err := sequence.New(steps, req)
		if err != nil {
			ctx.Log.Error(errors.Wrap(err, "failed to sequence.New"))
			return vk.E(http.StatusInternalServerError, "failed to handle request")
		}

		dispatcher := newDispatcher(ctx.Log, s.bus.Connect(), seq)

		if err := dispatcher.Execute(); err != nil {
			ctx.Log.Error(errors.Wrap(err, "failed to Execute"))
			return vk.E(http.StatusInternalServerError, "failed to execute plugin")
		}

		// handle any response headers that were set by the Runnables.
		if req.RespHeaders != nil {
			for head, val := range req.RespHeaders {
				w.Header().Add(head, val)
			}
		}

		responseData := seq.Request().State[mod.FQMN]

		return vk.RespondBytes(ctx.Context, w, responseData, http.StatusOK)
	}
}

func (s *Server) executePluginByRefHandler() vk.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, ctx *vk.Ctx) error {
		ref := readParam(ctx, "ref")

		mod := s.syncer.GetModuleByRef(ref)
		if mod == nil {
			return vk.E(http.StatusNotFound, "module not found")
		}

		ctx.Log.Debug("found module by ref:", mod.FQMN)

		req, err := request.FromVKRequest(r, ctx)
		if err != nil {
			ctx.Log.Error(errors.Wrap(err, "failed to request.FromVKRequest"))
			return vk.E(http.StatusInternalServerError, "failed to handle request")
		}

		req.UseSuborbitalHeaders(r, ctx)

		steps := []tenant.WorkflowStep{{FQMN: mod.FQMN}}

		// a sequence executes the handler's steps and manages its state.
		seq, err := sequence.New(steps, req)
		if err != nil {
			ctx.Log.Error(errors.Wrap(err, "failed to sequence.New"))
			return vk.E(http.StatusInternalServerError, "failed to handle request")
		}

		dispatcher := newDispatcher(ctx.Log, s.bus.Connect(), seq)

		if err := dispatcher.Execute(); err != nil {
			ctx.Log.Error(errors.Wrap(err, "failed to Execute"))
			return vk.E(http.StatusInternalServerError, "failed to execute plugin")
		}

		// handle any response headers that were set by the Runnables.
		if req.RespHeaders != nil {
			for head, val := range req.RespHeaders {
				w.Header().Add(head, val)
			}
		}

		responseData := seq.Request().State[mod.FQMN]

		return vk.RespondBytes(ctx.Context, w, responseData, http.StatusOK)
	}
}

func (s *Server) executeWorkflowHandler() vk.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, ctx *vk.Ctx) error {
		ident := readParam(ctx, "ident")
		namespace := readParam(ctx, "namespace")
		name := readParam(ctx, "name")

		tnt := s.syncer.TenantOverview(ident)
		if tnt == nil {
			ctx.Log.Error(fmt.Errorf("failed to find tenant %s", ident))
			return vk.E(http.StatusNotFound, "not found")
		}

		namespaces := []tenant.NamespaceConfig{tnt.Config.DefaultNamespace}
		namespaces = append(namespaces, tnt.Config.Namespaces...)

		var workflow *tenant.Workflow

		// yes, this is a dumb and slow way to do this but we'll optimize later

	OUTER:
		for i := range namespaces {
			ns := namespaces[i]
			if ns.Name != namespace {
				continue
			}

			for j := range ns.Workflows {
				wfl := ns.Workflows[j]

				if wfl.Name != name {
					continue
				}

				workflow = &wfl
				break OUTER
			}
		}

		if workflow == nil {
			ctx.Log.Error(fmt.Errorf("failed to find workflow %s", ident))
			return vk.E(http.StatusNotFound, "not found")
		}

		req, err := request.FromVKRequest(r, ctx)
		if err != nil {
			ctx.Log.Error(errors.Wrap(err, "failed to request.FromVKRequest"))
			return vk.E(http.StatusInternalServerError, "failed to handle request")
		}

		req.UseSuborbitalHeaders(r, ctx)

		// a sequence executes the handler's steps and manages its state.
		seq, err := sequence.New(workflow.Steps, req)
		if err != nil {
			ctx.Log.Error(errors.Wrap(err, "failed to sequence.New"))
			return vk.E(http.StatusInternalServerError, "failed to handle request")
		}

		dispatcher := newDispatcher(ctx.Log, s.bus.Connect(), seq)

		if err := dispatcher.Execute(); err != nil {
			ctx.Log.Error(errors.Wrap(err, "failed to Execute"))
			return vk.E(http.StatusInternalServerError, "failed to execute plugin")
		}

		// handle any response headers that were set by the Runnables.
		if req.RespHeaders != nil {
			for head, val := range req.RespHeaders {
				w.Header().Add(head, val)
			}
		}

		// this should be smarter eventually (i.e. handle last-step groups properly)
		responseData := seq.Request().State[workflow.Steps[len(workflow.Steps)-1].FQMN]

		return vk.RespondBytes(ctx.Context, w, responseData, http.StatusOK)
	}
}

func (s *Server) healthHandler() vk.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, ctx *vk.Ctx) error {
		return vk.RespondJSON(ctx.Context, w, map[string]bool{"healthy": true}, http.StatusOK)
	}
}

func readParam(ctx *vk.Ctx, name string) string {
	v := ctx.Get(name)
	if v != nil {
		return v.(string)
	}

	return ctx.Params.ByName(name)
}
