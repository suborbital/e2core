package orchestrator

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"runtime"
	"sync"
	"syscall"
	"text/template"
	"time"

	"github.com/pkg/errors"
	"github.com/sethvargo/go-envconfig"

	"github.com/suborbital/deltav/server/appsource"
	"github.com/suborbital/vektor/vlog"

	"github.com/suborbital/deltav/orchestrator/config"
	"github.com/suborbital/deltav/orchestrator/exec"
)

const (
	atmoPort = "8080"
)

type Orchestrator struct {
	logger     *vlog.Logger
	config     config.Config
	sats       map[string]*watcher // map of FQFNs to watchers
	signalChan chan os.Signal
	wg         sync.WaitGroup
}

type commandTemplateData struct {
	Port string
}

func New(bundlePath string) (*Orchestrator, error) {
	conf, err := config.Parse(bundlePath, envconfig.OsLookuper())
	if err != nil {
		return nil, errors.Wrap(err, "failed to config.Parse")
	}

	l := vlog.Default(
		vlog.EnvPrefix("DELTAV"),
		vlog.Level(vlog.LogLevelWarn),
	)

	o := &Orchestrator{
		logger: l,
		config: conf,
		sats:   map[string]*watcher{},
		wg:     sync.WaitGroup{},
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
			break
		}

		o.reconcileConstellation(appSource, errChan)

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

func (o *Orchestrator) RunPartner(command string) error {
	o.logger.Debug("starting partner:", command)

	data := commandTemplateData{
		Port: "3000",
	}

	addr, exists := os.LookupEnv("DELTAV_PARTNER")
	if exists {
		partnerUrl, err := url.Parse(addr)
		if err != nil {
			return errors.Wrap(err, "failed to Parse")
		}

		data.Port = partnerUrl.Port()
	}

	tpl := template.New("cmd")
	tpl.Parse(command)

	out := bytes.NewBuffer(nil)
	if err := tpl.Execute(out, data); err != nil {
		return errors.Wrap(err, "failed to Execute command template")
	}

	if _, _, err := exec.Run(out.String(), fmt.Sprintf("PORT=%s", data.Port)); err != nil {
		return errors.Wrap(err, "failed to Run")
	}

	return nil
}

func (o *Orchestrator) reconcileConstellation(appSource appsource.AppSource, errChan chan error) {
	apps := appSource.Applications()

	for _, app := range apps {
		runnables := appSource.Runnables(app.Identifier, app.AppVersion)

		for i := range runnables {
			runnable := runnables[i]

			o.logger.Debug("reconciling", runnable.FQFN)

			if _, exists := o.sats[runnable.FQFN]; !exists {
				o.sats[runnable.FQFN] = newWatcher(runnable.FQFN, o.logger)
			}

			satWatcher := o.sats[runnable.FQFN]

			launch := func() {
				o.logger.Debug("launching sat (", runnable.FQFN, ")")

				cmd, port := satCommand(o.config, runnable)

				// repeat forever in case the command does error out
				uuid, pid, err := exec.Run(
					cmd,
					"SAT_HTTP_PORT="+port,
					"SAT_ENV_TOKEN="+o.config.EnvToken,
					"SAT_CONTROL_PLANE="+o.config.ControlPlane,
				)

				if err != nil {
					o.logger.Error(errors.Wrapf(err, "failed to exec.Run sat ( %s )", runnable.FQFN))
					return
				}

				satWatcher.add(runnable.FQFN, port, uuid, pid)

				o.logger.Debug("successfully started sat (", runnable.FQFN, ") on port", port)
			}

			// we want to max out at 8 threads per instance
			threshold := runtime.NumCPU() / 2
			if threshold > 8 {
				threshold = 8
			}

			report := satWatcher.report()

			if report == nil || report.instCount == 0 {
				// if no instances exist, launch one
				o.logger.Debug("no instances exist for", runnable.FQFN)

				go launch()
			} else if report.instCount > 0 && report.totalThreads/report.instCount >= threshold {
				if report.instCount >= runtime.NumCPU() {
					o.logger.Warn("maximum instance count reached for", runnable.Name)
				} else {
					// if the current instances seem overwhelmed, add one
					o.logger.Debug("scaling up", runnable.Name, "; totalThreads:", report.totalThreads, "instCount:", report.instCount)

					go launch()
				}
			} else if report.instCount > 0 && report.totalThreads/report.instCount < threshold {
				if report.instCount == 1 {
					// that's fine, do nothing
				} else {
					// if the current instances have too much spare time on their hands
					o.logger.Debug("scaling down", runnable.Name, "; totalThreads:", report.totalThreads, "instCount:", report.instCount)

					satWatcher.scaleDown()
				}
			}

			if report != nil {
				for _, p := range report.failedPorts {
					o.logger.Debug("killing instance from failed port", p)

					satWatcher.terminateInstance(p)
				}
			}
		}
	}
}

func (o *Orchestrator) setupAppSource() (appsource.AppSource, chan error) {
	// if an external control plane hasn't been set, act as the control plane
	// but if one has been set, use it (and launch all children with it configured)
	if o.config.ControlPlane == config.DefaultControlPlane {
		appSource, errChan := startAppSourceServer(o.config.BundlePath)
		return appSource, errChan
	}

	appSource := appsource.NewHTTPSource(o.config.ControlPlane)

	if err := startAppSourceWithRetry(o.logger, appSource); err != nil {
		log.Fatal(errors.Wrap(err, "failed to startAppSourceHTTPClient"))
	}

	if err := registerWithControlPlane(o.config); err != nil {
		log.Fatal(errors.Wrap(err, "failed to registerWithControlPlane"))
	}

	errChan := make(chan error)

	return appSource, errChan
}
