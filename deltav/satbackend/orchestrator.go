package satbackend

import (
	"context"
	"log"
	"os"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/pkg/errors"

	"github.com/suborbital/appspec/appsource"
	"github.com/suborbital/appspec/appsource/client"
	"github.com/suborbital/deltav/deltav/satbackend/exec"
	"github.com/suborbital/deltav/options"
	"github.com/suborbital/vektor/vlog"
)

const (
	atmoPort = "8080"
)

type Orchestrator struct {
	logger           *vlog.Logger
	opts             options.Options
	sats             map[string]*watcher // map of FQFNs to watchers
	failedPortCounts map[string]int
	signalChan       chan os.Signal
	wg               sync.WaitGroup
}

func New(bundlePath string, opts options.Options) (*Orchestrator, error) {
	o := &Orchestrator{
		logger:           opts.Logger(),
		opts:             opts,
		sats:             map[string]*watcher{},
		failedPortCounts: map[string]int{},
		wg:               sync.WaitGroup{},
	}

	return o, nil
}

func (o *Orchestrator) Start(ctx context.Context) error {
	appSource, errChan := o.setupAppSource()

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

		o.reconcileConstellation(appSource)

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

func (o *Orchestrator) reconcileConstellation(appSource appsource.AppSource) {
	ovv, err := appSource.Overview()
	if err != nil {
		o.logger.Error(errors.Wrap(err, "failed to app.Overview"))
	}

	// mount each handler into the VK group.
	for ident, version := range ovv.TenantRefs.Identifiers {
		tnt, err := appSource.TenantOverview(ident)
		if err != nil {
			o.logger.Error(errors.Wrapf(err, "failed to app.TenantOverview for %s", ident))
			return
		}

		if tnt.Version != version {
			o.logger.Warn("encountered version mismatch for tenant", ident, "expected", version, "got", tnt.Version)
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
					"SAT_ENV_TOKEN="+o.opts.EnvironmentToken,
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
func (o *Orchestrator) setupAppSource() (appsource.AppSource, chan error) {
	// if an external control plane hasn't been set, act as the control plane
	// but if one has been set, use it (and launch all children with it configured)
	if o.opts.ControlPlane == options.DefaultControlPlane || o.opts.ControlPlane == "" {
		o.opts.ControlPlane = options.DefaultControlPlane

		o.logger.Debug("starting AppSource server")

		// the returned appSource is a bundleSource
		appSource, errChan := startAppSourceServer(o.opts.BundlePath)

		return appSource, errChan
	}

	o.logger.Debug("using passthrough AppSource client")

	appSource := client.NewHTTPSource(o.opts.ControlPlane, nil)

	if err := startAppSourceWithRetry(o.logger, appSource); err != nil {
		log.Fatal(errors.Wrap(err, "failed to startAppSourceHTTPClient"))
	}

	if err := registerWithControlPlane(o.opts); err != nil {
		log.Fatal(errors.Wrap(err, "failed to registerWithControlPlane"))
	}

	errChan := make(chan error)

	return appSource, errChan
}
