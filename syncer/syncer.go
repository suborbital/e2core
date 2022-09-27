package syncer

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"

	"github.com/suborbital/appspec/system"
	"github.com/suborbital/appspec/tenant"
	"github.com/suborbital/e2core/options"
	"github.com/suborbital/e2core/scheduler"
	"github.com/suborbital/vektor/vlog"
)

// Syncer keeps an in-memory cache of the system state such that the coordinator and orchestrator
// can get up-to-date information about the world.
type Syncer struct {
	sched *scheduler.Scheduler
	job   *syncJob
	opts  *options.Options
}

type syncJob struct {
	appSource    system.Source
	state        *system.State
	tenantIdents map[string]int64
	overviews    map[string]system.TenantOverview
	modules      map[string]tenant.Module

	log  *vlog.Logger
	lock *sync.RWMutex
}

// New creates a syncer with the given AppSource
func New(opts *options.Options, source system.Source) *Syncer {
	s := &Syncer{
		sched: scheduler.New(),
		opts:  opts,
	}

	s.job = &syncJob{
		appSource:    source,
		state:        &system.State{},
		tenantIdents: make(map[string]int64),
		overviews:    make(map[string]system.TenantOverview),
		modules:      make(map[string]tenant.Module),
		log:          opts.Logger(),
		lock:         &sync.RWMutex{},
	}

	s.sched.Register("sync", s.job)

	return s
}

// Start starts the syncer
func (s *Syncer) Start() error {
	if err := s.job.appSource.Start(s.opts); err != nil {
		return errors.Wrap(err, "failed to appSource.Start")
	}

	// sync once to seed the initial state
	if _, err := s.sched.Do(scheduler.NewJob("sync", nil)).Then(); err != nil {
		return errors.Wrap(err, "failed to Do sync job")
	}

	s.sched.Schedule(scheduler.Every(1, func() scheduler.Job { return scheduler.NewJob("sync", nil) }))

	return nil
}

// Run runs a sync job
func (s *syncJob) Run(job scheduler.Job, ctx *scheduler.Ctx) (interface{}, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	state, err := s.appSource.State()
	if err != nil {
		return nil, errors.Wrap(err, "failed to appSource.State")
	}

	if state.SystemVersion == s.state.SystemVersion {
		s.log.Debug(fmt.Sprintf("skipping sync with version match: %d, %d", state.SystemVersion, s.state.SystemVersion))
		return nil, nil
	}

	s.log.Debug(fmt.Sprintf("running sync with version mismatch: %d, %d", state.SystemVersion, s.state.SystemVersion))

	ovv, err := s.appSource.Overview()
	if err != nil {
		return nil, errors.Wrap(err, "failed to app.Overview")
	}

	// mount each handler into the VK group.
	for ident, version := range ovv.TenantRefs.Identifiers {
		localTnt, exists := s.overviews[ident]
		if exists && localTnt.Version == version {
			continue
		}

		tnt, err := s.appSource.TenantOverview(ident)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to app.TenantOverview for %s", ident)
		}

		s.overviews[ident] = *tnt

		s.log.Debug("syncing", len(tnt.Config.Modules), "modules for", ident)

		for i, m := range tnt.Config.Modules {
			s.log.Debug("syncing module:", m.Ref, m.Name, m.Namespace)

			s.modules[m.Ref] = tnt.Config.Modules[i]
		}

		s.log.Debug("synced tenant", ident, "to version", version)
	}

	s.log.Debug("completed sync at version", state.SystemVersion)

	s.state = state
	s.tenantIdents = ovv.TenantRefs.Identifiers

	return nil, nil
}

func (s *syncJob) OnChange(_ scheduler.ChangeEvent) error { return nil }

// State returns the current system state
func (s *Syncer) State() *system.State {
	s.job.lock.RLock()
	defer s.job.lock.RUnlock()

	return s.job.state
}

// ListTenants returns a map of tenant idents to their latest versions
func (s *Syncer) ListTenants() map[string]int64 {
	s.job.lock.RLock()
	defer s.job.lock.RUnlock()

	return s.job.tenantIdents
}

// TenantOverview returns the (possibly nil) TenantOverview for the given tenant ident
func (s *Syncer) TenantOverview(ident string) *system.TenantOverview {
	s.job.lock.RLock()
	defer s.job.lock.RUnlock()

	ovv := s.job.overviews[ident]

	return &ovv
}

// GetModuleByName gets a module by its name
func (s *Syncer) GetModuleByName(ident, namespace, name string) *tenant.Module {
	s.job.lock.RLock()
	defer s.job.lock.RUnlock()

	tnt := s.TenantOverview(ident)
	if tnt == nil {
		return nil
	}

	var mod *tenant.Module

	for i, m := range tnt.Config.Modules {
		if m.Namespace == namespace && m.Name == name {
			mod = &tnt.Config.Modules[i]
			break
		}
	}

	return mod
}

// GetModuleByRef gets a module by its ref
func (s *Syncer) GetModuleByRef(ref string) *tenant.Module {
	s.job.lock.RLock()
	defer s.job.lock.RUnlock()

	mod := s.job.modules[ref]

	return &mod
}
