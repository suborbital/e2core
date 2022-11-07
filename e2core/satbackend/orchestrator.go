package satbackend

import (
	"context"
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
		wg:               sync.WaitGroup{},
	}

	return o, nil
}

func (o *Orchestrator) Start(ctx context.Context) error {
	if err := o.syncer.Start(); err != nil {
		return errors.Wrap(err, "failed to syncer.Start")
	}

	errChan := o.setupSystemSourceServer()

	o.wg.Add(1)

loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		case err := <-errChan:
			return err
		default:
			// fall through and reconcile
		}

		o.reconcileConstellation(o.syncer)

		time.Sleep(time.Second)
	}

	o.logger.Debug("stopping orchestrator")

	for _, s := range o.sats {
		err := s.terminate()
		if err != nil {
			log.Fatal("terminating sats failed", err)
		}
	}

	o.wg.Done()

	return nil
}

// Shutdown signals to the orchestrator that shutdown is needed
// mostly only required for testing purposes as the OS handles it normally
func (o *Orchestrator) Shutdown() {
	o.signalChan <- syscall.SIGTERM

	o.wg.Wait()
}

func (o *Orchestrator) reconcileConstellation(syncer *syncer.Syncer) {
	tenants := syncer.ListTenants()
	if tenants == nil {
		o.logger.ErrorString("tenants is nil")
	}

	// mount each handler into the VK group.
	for ident := range tenants {
		tnt := syncer.TenantOverview(ident)
		if tnt == nil {
			o.logger.ErrorString(fmt.Sprintf("failed to syncer.TenantOverview for %s", ident))
			continue
		}

		for i := range tnt.Config.Modules {
			module := tnt.Config.Modules[i]

			o.logger.Debug("reconciling", module.FQMN)

			if _, exists := o.sats[module.FQMN]; !exists {
				o.sats[module.FQMN] = newWatcher(module.FQMN, o.logger)
			}

			satWatcher := o.sats[module.FQMN]

			launch := func() {
				o.logger.Debug("launching sat (", module.FQMN, ")")

				cmd, port := modStartCommand(module)

				// repeat forever in case the command does error out
				uuid, pid, err := exec.Run(
					cmd,
					"SAT_HTTP_PORT="+port,
					"SAT_CONTROL_PLANE="+o.opts.ControlPlane,
				)

				if err != nil {
					o.logger.Error(errors.Wrapf(err, "failed to exec.Run sat ( %s )", module.FQMN))
					return
				}

				satWatcher.add(module.FQMN, port, uuid, pid)

				o.logger.Debug("successfully started sat (", module.FQMN, ") on port", port)
			}

			// we want to max out at 8 threads per instance
			threshold := runtime.NumCPU() / 2
			if threshold > 8 {
				threshold = 8
			}

			report := satWatcher.report()

			if report == nil || report.instCount == 0 {
				// if no instances exist, launch one
				o.logger.Debug("no instances exist for", module.FQMN)

				go launch()
			} else if report.instCount > 0 && report.totalThreads/report.instCount >= threshold {
				if report.instCount >= runtime.NumCPU() {
					o.logger.Warn("maximum instance count reached for", module.Name)
				} else {
					// if the current instances seem overwhelmed, add one
					o.logger.Debug("scaling up", module.Name, "; totalThreads:", report.totalThreads, "instCount:", report.instCount)

					go launch()
				}
			} else if report.instCount > 0 && report.totalThreads/report.instCount < threshold {
				if report.instCount == 1 {
					// that's fine, do nothing
				} else {
					// if the current instances have too much spare time on their hands
					o.logger.Debug("scaling down", module.Name, "; totalThreads:", report.totalThreads, "instCount:", report.instCount)

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
						o.logger.Debug("killing instance from failed port", p)

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

		o.logger.Debug("starting SystemSource server")

		errChan := startSystemSourceServer(o.opts.BundlePath)

		return errChan
	}

	o.logger.Debug("registering with control plane")

	if err := registerWithControlPlane(*o.opts); err != nil {
		log.Fatal(errors.Wrap(err, "failed to registerWithControlPlane"))
	}

	errChan := make(chan error)

	return errChan
}
