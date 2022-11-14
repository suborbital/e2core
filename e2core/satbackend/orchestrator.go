package satbackend

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/pkg/errors"

	"github.com/suborbital/e2core/e2core/satbackend/exec"
	"github.com/suborbital/e2core/options"
	"github.com/suborbital/e2core/syncer"
	"github.com/suborbital/vektor/vlog"
)

const (
	atmoPort = "8080"
)

type Orchestrator struct {
	syncer           *syncer.Syncer
	logger           *vlog.Logger
	opts             *options.Options
	sats             map[string]*watcher // map of FQFNs to watchers
	failedPortCounts map[string]int
	signalChan       chan os.Signal
	wg               sync.WaitGroup
}

func New(opts *options.Options, syncer *syncer.Syncer) (*Orchestrator, error) {
	o := &Orchestrator{
		syncer:           syncer,
		logger:           opts.Logger(),
		opts:             opts,
		sats:             map[string]*watcher{},
		failedPortCounts: map[string]int{},
		signalChan:       make(chan os.Signal),
		wg:               sync.WaitGroup{},
	}

	return o, nil
}

func (o *Orchestrator) Start(ctx context.Context) error {
	if err := o.syncer.Start(); err != nil {
		return errors.Wrap(err, "failed to syncer.Start")
	}

	errChan := o.setupSystemSourceServer()

	o.logger.Warn("[orchestrator.start] adding one to delta")
	o.wg.Add(1)

	var err error

	o.logger.Warn("[orchestrator.start] creating a ticker")
	ticker := time.NewTicker(time.Second)
loop:
	for {
		select {
		case <-ctx.Done():
			o.logger.Warn("[orchestrator.start] ctx done called, breaking out of loop")
			// if context timeout reached or we manually cancelled the context
			break loop

		case <-o.signalChan:
			o.logger.Warn("[orchestrator.start] something fron signal chan")
			// if anything gets sent in the signal channel
			break loop

		case err = <-errChan:
			o.logger.Warn("[orchestrator.start] something from error chan")
			// if there's an error
			break loop

		case <-ticker.C:
			o.logger.Warn("[orchestrator.start] a message from ticker")
			// each second do this
			o.reconcileConstellation(o.syncer)
		}
	}

	o.logger.Debug("[orchestrator.start] stopping orchestrator")

	for _, s := range o.sats {
		o.logger.Debug("[orchestrator.start] terminating sat instance")
		err := s.terminate()
		if err != nil {
			log.Fatal("[orchestrator.start] terminating sats failed", err)
		}
	}

	o.wg.Done()

	return err
}

// Shutdown signals to the orchestrator that shutdown is needed
// mostly only required for testing purposes as the OS handles it normally
func (o *Orchestrator) Shutdown() {
	o.logger.Debug("[orchestrator.Shutdown] sending sigterm")
	o.signalChan <- syscall.SIGTERM

	o.logger.Debug("[orchestrator.Shutdown] waiting")

	o.wg.Wait()
}

func (o *Orchestrator) reconcileConstellation(syncer *syncer.Syncer) {
	o.logger.Warn("[orchestrator.reconcileConstellation] reconciling...")
	tenants := syncer.ListTenants()
	if tenants == nil {
		o.logger.ErrorString("[orchestrator.reconcileConstellation] tenants is nil")
	}

	// mount each handler into the VK group.
	for ident := range tenants {
		tnt := syncer.TenantOverview(ident)
		if tnt == nil {
			o.logger.ErrorString(fmt.Sprintf("[orchestrator.reconcileConstellation] failed to syncer.TenantOverview for %s", ident))
			continue
		}

		defaultConnectionsJSON, err := json.Marshal(tnt.Config.DefaultNamespace.Connections)
		if err != nil {
			o.logger.ErrorString("failed to json.Marshal Connections config, will continue")
		}

		for i := range tnt.Config.Modules {
			module := tnt.Config.Modules[i]

			connectionsEnv := ""
			if module.Namespace == "default" {
				connectionsEnv = string(defaultConnectionsJSON)
			}

			o.logger.Debug("[orchestrator.reconcileConstellation] reconciling", module.FQMN)

			if _, exists := o.sats[module.FQMN]; !exists {
				o.sats[module.FQMN] = newWatcher(module.FQMN, o.logger)
			}

			satWatcher := o.sats[module.FQMN]

			launch := func() {
				o.logger.Debug("[orchestrator.reconcileConstellation] launching sat (", module.FQMN, ")")

				cmd, port := modStartCommand(module)

				// repeat forever in case the command does error out
				uuid, pid, err := exec.Run(
					cmd,
					"SAT_HTTP_PORT="+port,
					"SAT_CONTROL_PLANE="+o.opts.ControlPlane,
					"SAT_CONNECTIONS"+connectionsEnv,
				)

				if err != nil {
					o.logger.Error(errors.Wrapf(err, "[orchestrator.reconcileConstellation] failed to exec.Run sat ( %s )", module.FQMN))
					return
				}

				satWatcher.add(module.FQMN, port, uuid, pid)

				o.logger.Debug("[orchestrator.reconcileConstellation] successfully started sat (", module.FQMN, ") on port", port)
			}

			// we want to max out at 8 threads per instance
			threshold := runtime.NumCPU() / 2
			if threshold > 8 {
				threshold = 8
			}

			report := satWatcher.report()

			if report == nil || report.instCount == 0 {
				// if no instances exist, launch one
				o.logger.Debug("[orchestrator.reconcileConstellation] no instances exist for", module.FQMN)

				go launch()
			} else if report.instCount > 0 && report.totalThreads/report.instCount >= threshold {
				if report.instCount >= runtime.NumCPU() {
					o.logger.Warn("[orchestrator.reconcileConstellation] maximum instance count reached for", module.Name)
				} else {
					// if the current instances seem overwhelmed, add one
					o.logger.Debug("[orchestrator.reconcileConstellation] scaling up", module.Name, "; totalThreads:", report.totalThreads, "instCount:", report.instCount)

					go launch()
				}
			} else if report.instCount > 0 && report.totalThreads/report.instCount < threshold {
				if report.instCount == 1 {
					// that's fine, do nothing
				} else {
					// if the current instances have too much spare time on their hands
					o.logger.Debug("[orchestrator.reconcileConstellation] scaling down", module.Name, "; totalThreads:", report.totalThreads, "instCount:", report.instCount)

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
						o.logger.Debug("[orchestrator.reconcileConstellation] killing instance from failed port", p)

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

// TODO: implement and use an authSource when creating NewHTTPSource
func (o *Orchestrator) setupSystemSourceServer() chan error {
	// if an external control plane hasn't been set, act as the control plane
	// but if one has been set, use it (and launch all children with it configured)
	if o.opts.ControlPlane == options.DefaultControlPlane || o.opts.ControlPlane == "" {
		o.opts.ControlPlane = options.DefaultControlPlane

		o.logger.Debug("[orchestrator.setupSystemSourceServer] starting SystemSource server")

		errChan := startSystemSourceServer(o.opts.BundlePath)

		return errChan
	}

	o.logger.Debug("[orchestrator.setupSystemSourceServer] registering with control plane")

	if err := registerWithControlPlane(*o.opts); err != nil {
		log.Fatal(errors.Wrap(err, "[orchestrator.setupSystemSourceServer] failed to registerWithControlPlane"))
	}

	errChan := make(chan error)

	return errChan
}
