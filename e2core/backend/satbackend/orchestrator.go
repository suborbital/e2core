package satbackend

import (
	"encoding/json"
	"os"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/suborbital/e2core/e2core/backend/satbackend/exec"
	"github.com/suborbital/e2core/e2core/options"
	"github.com/suborbital/e2core/e2core/syncer"
)

type Orchestrator struct {
	syncer           *syncer.Syncer
	logger           zerolog.Logger
	opts             *options.Options
	sats             map[string]*watcher // map of FQMNs to watchers
	failedPortCounts map[string]int
	signalChan       chan os.Signal
	wg               sync.WaitGroup
}

func New(logger zerolog.Logger, opts *options.Options, syncer *syncer.Syncer) (*Orchestrator, error) {
	o := &Orchestrator{
		syncer:           syncer,
		logger:           logger.With().Str("module", "orchestrator").Logger(),
		opts:             opts,
		sats:             map[string]*watcher{},
		failedPortCounts: map[string]int{},
		signalChan:       make(chan os.Signal),
		wg:               sync.WaitGroup{},
	}

	return o, nil
}

func (o *Orchestrator) Start() error {
	if err := o.syncer.Start(); err != nil {
		return errors.Wrap(err, "failed to syncer.Start")
	}

	ll := o.logger.With().Str("method", "Start").Logger()

	o.wg.Add(1)

	var err error

	ticker := time.NewTicker(time.Second)
loop:
	for {
		select {
		case <-o.signalChan:
			// if anything gets sent in the signal channel
			ll.Warn().Msg("received on signal chan")
			break loop

		case <-ticker.C:
			// when the ticker fires each second
			o.reconcileConstellation(o.syncer)
		}
	}

	ll.Debug().Msg("stopping orchestrator")

	for _, s := range o.sats {
		ll.Debug().Str("satFQMN", s.fqmn).Msg("terminating sat instance")
		s.terminate()
	}

	o.wg.Done()

	return err
}

// Shutdown signals to the orchestrator that shutdown is needed
// mostly only required for testing purposes as the OS handles it normally
func (o *Orchestrator) Shutdown() {
	ll := o.logger.With().Str("method", "Shutdown").Logger()

	ll.Debug().Msg("sending sigterm")
	o.signalChan <- syscall.SIGTERM

	ll.Debug().Msg("waiting")
	o.wg.Wait()

	ll.Debug().Msg("shutdown completed")
}

func (o *Orchestrator) reconcileConstellation(syncer *syncer.Syncer) {
	ll := o.logger.With().Str("method", "reconcileConstellation").Logger()

	ll.Debug().Msg("reconciling...")

	tenants := syncer.ListTenants()
	if tenants == nil {
		ll.Error().Msg("tenants is nil")
	}

	// mount each handler into the handler group.
	for ident := range tenants {
		tnt := syncer.TenantOverview(ident)
		if tnt == nil {
			ll.Error().Str("ident", ident).Msg("syncer.TenantOverview is nil")
			continue
		}

		defaultConnectionsJSON, err := json.Marshal(tnt.Config.DefaultNamespace.Connections)
		if err != nil {
			ll.Err(err).Msg("json.Marshal default connections, will continue")
		}

		for i := range tnt.Config.Modules {
			module := tnt.Config.Modules[i]

			ll.Debug().Str("moduleFQMN", module.FQMN).Msg("reconciling")

			if _, exists := o.sats[module.FQMN]; !exists {
				o.sats[module.FQMN] = newWatcher(module.FQMN, o.logger)
			}

			satWatcher := o.sats[module.FQMN]

			ll.Info().Msg("starting the range over the things in the died channel")

			satWatcher.diedListLock.Lock()
			for diedPort := range satWatcher.diedList {
				_ = satWatcher.terminateInstance(diedPort)
			}
			satWatcher.diedList = make(map[string]struct{})
			satWatcher.diedListLock.Unlock()
			ll.Info().Msg("no more things, continuing reconciliation")

			launch := func() {
				cmd, port := modStartCommand(module)

				ll.Debug().Str("moduleFQMN", module.FQMN).Str("port", port).Msg("launching sat")

				connectionsEnv := ""
				if module.Namespace == "default" {
					connectionsEnv = string(defaultConnectionsJSON)
				}

				// repeat forever in case the command does error out
				processUUID, cxl, wait, err := exec.Run(
					cmd,
					"SAT_HTTP_PORT="+port,
					"SAT_CONTROL_PLANE="+o.opts.ControlPlane,
					"SAT_CONNECTIONS="+connectionsEnv,
				)
				if err != nil {
					ll.Err(err).Str("moduleFQMN", module.FQMN).Msg("exec.Run failed for sat instance")
					return
				}

				go func() {
					err := wait()
					if err != nil {
						ll.Err(err).Str("moduleFQMN", module.FQMN).Str("port", port).Msg("calling waitfunc for the module failed")
					}

					err = satWatcher.addDied(port)
					if err != nil {
						ll.Err(err).Str("moduleFQMN", module.FQMN).Str("port", port).Msg("adding the port to the died list")
					}

					ll.Info().Str("moduleFQMN", module.FQMN).Str("port", port).Msg("sent died message into channel")
				}()

				satWatcher.add(module.FQMN, port, processUUID, cxl)

				ll.Debug().Str("moduleFQMN", module.FQMN).Str("port", port).Msg("successfully started sat")
			}

			// we want to max out at 8 threads per instance
			threshold := runtime.NumCPU() / 2
			if threshold > 8 {
				threshold = 8
			}

			report := satWatcher.report()

			if report == nil || report.instCount == 0 {
				// if no instances exist, launch one
				ll.Debug().Str("moduleFQMN", module.FQMN).Msg("no instance exists")

				go launch()
			} else if report.instCount > 0 && report.totalThreads/report.instCount >= threshold {
				if report.instCount >= runtime.NumCPU() {
					ll.Warn().Str("moduleName", module.Name).Msg("maximum instance count reached for named modules")
				} else {
					// if the current instances seem overwhelmed, add one
					ll.Debug().
						Str("moduleName", module.Name).
						Int("totalThreads", report.totalThreads).
						Int("instanceCount", report.instCount).
						Msg("scaling up")

					go launch()
				}
			} else if report.instCount > 0 && report.totalThreads/report.instCount < threshold {
				if report.instCount == 1 {
					// that's fine, do nothing
				} else {
					// if the current instances have too much spare time on their hands
					ll.Debug().
						Str("moduleName", module.Name).
						Int("totalThreads", report.totalThreads).
						Int("instanceCount", report.instCount).
						Msg("scaling down")

					satWatcher.scaleDown()
				}
			}

			if report != nil {
				// for each failed port, track how many times it's failed and terminate if > 5
				for _, p := range report.failedPorts {
					count, exists := o.failedPortCounts[p]
					if !exists {
						o.failedPortCounts[p] = 1
					} else if count > 5 {
						ll.Debug().Str("port", p).Msg("killing instance from failed port")

						satWatcher.terminateInstance(p)

						delete(o.failedPortCounts, p)
					} else {
						o.failedPortCounts[p] = count + 1
					}
				}
			}
		}
	}
}
