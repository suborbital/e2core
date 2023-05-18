package overviews

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/suborbital/e2core/foundation/tracing"
	"github.com/suborbital/systemspec/system"
)

const (
	systemOverviewMask = "%s/system/v1/overview"
	tenantOverviewMask = "%s/system/v1/tenant/%s"
	clientTimeout      = time.Second
)

type tenantID string
type namespaceFunctionName string
type moduleRef string

type Repository struct {
	endpoint string
	data     map[tenantID]tenantData
	shutdown chan struct{}
	logger   zerolog.Logger
	client   *http.Client
	lock     *sync.Mutex
	wg       *sync.WaitGroup
}

type Config struct {
	Endpoint string
}

type tenantData map[namespaceFunctionName]moduleRef

func NewRepository(c Config, l zerolog.Logger) *Repository {
	return &Repository{
		endpoint: c.Endpoint,
		data:     make(map[tenantID]tenantData),
		shutdown: make(chan struct{}),
		logger:   l.With().Str("component", "repository").Logger(),
		client:   &http.Client{Timeout: clientTimeout},
		lock:     new(sync.Mutex),
		wg:       new(sync.WaitGroup),
	}
}

func (r *Repository) Start() {
	go r.work()
}

// work will keep an up-to-date repository of data from control plane regarding available modules, their refs, for each
// tenant.
//
// 1. Each second it will ask the control plane for the system overview which will get us a list of tenants and how many
// modules it has.
// 2. Then for each tenant that have more than 0 modules, it will ask for the tenant overview.
// 3. With the tenant overview it will iterate over the individual modules, and store the data in a map.
// 4. Lastly it will replace the data about the modules and refs in its own data store.
func (r *Repository) work() {
	t := time.NewTicker(5 * time.Second)
	r.wg.Add(1)

	for {
		select {
		case <-t.C:
			ctx, span := tracing.Tracer.Start(context.Background(), "repository.work.tick")

			so, err := r.systemOverview(ctx)
			if err != nil {
				r.logger.Err(err).Msg("r.systemOverview")
				span.End()
				continue
			}

			d := make(map[tenantID]tenantData)

			// Iterate over the tenants.
			for tid, numberOfModules := range so.TenantRefs.Identifiers {
				if numberOfModules == 0 {
					// If the tenant has 0 modules, skip it, because we don't need to keep track of module data.
					continue
				}

				d[tenantID(tid)] = make(tenantData)

				to, err := r.tenantOverview(ctx, tid)
				if err != nil {
					r.logger.Err(err).Str("tenant_id", tid).Msg("r.tenantOverview")
					continue
				}

				// Build the data.
				for _, mod := range to.Config.Modules {
					d[tenantID(tid)][namespaceFunctionName(fmt.Sprintf("%s-%s", mod.Namespace, mod.Name))] = moduleRef(mod.Ref)
				}
			}

			// swap out our data store
			r.lock.Lock()

			r.data = d

			r.lock.Unlock()

			r.logger.Info().Interface("repo", d).Msg("synced")

		case <-r.shutdown:
			r.wg.Done()
			return
		}
	}
}

func (r *Repository) systemOverview(ctx context.Context) (system.Overview, error) {
	ctx, span := tracing.Tracer.Start(ctx, "repository.systemOverview")
	defer span.End()
	// SystemOverview data grab.
	ctx, cxl := context.WithTimeout(ctx, time.Second)
	defer cxl()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf(systemOverviewMask, r.endpoint), nil)
	if err != nil {
		return system.Overview{}, errors.Wrap(err, "http.NewRequestWithContext")
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return system.Overview{}, errors.Wrap(err, "r.client.Do")
	}

	// Parse the system overview.
	var so system.Overview

	err = json.NewDecoder(resp.Body).Decode(&so)
	if err != nil {
		return system.Overview{}, errors.Wrap(err, "json.NewDecoder(res.Body).Decode")
	}

	return so, nil
}

func (r *Repository) tenantOverview(ctx context.Context, tenantID string) (system.TenantOverview, error) {
	ctx, span := tracing.Tracer.Start(ctx, "repository.tenantOverview")
	defer span.End()

	// Create context for the tenant overview request.
	ctx, cxl := context.WithTimeout(ctx, time.Second)
	defer cxl()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf(tenantOverviewMask, r.endpoint, tenantID), nil)
	if err != nil {
		return system.TenantOverview{}, errors.Wrap(err, "http.newRequestWithContext")
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return system.TenantOverview{}, errors.Wrap(err, "r.client.Do")
	}

	// Parse the tenant overview response.
	var to system.TenantOverview

	err = json.NewDecoder(resp.Body).Decode(&to)
	if err != nil {
		return system.TenantOverview{}, errors.Wrap(err, "json.NewDecoder(resp.Body).Decode")
	}

	return to, nil
}

func (r *Repository) Shutdown() {
	close(r.shutdown)

	r.wg.Wait()
}
