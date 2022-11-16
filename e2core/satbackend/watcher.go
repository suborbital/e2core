package satbackend

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"syscall"
	"time"

	"github.com/pkg/errors"

	"github.com/suborbital/e2core/e2core/satbackend/process"
	"github.com/suborbital/e2core/scheduler"
	"github.com/suborbital/vektor/vlog"
)

var httpClient = http.Client{Timeout: time.Second}

// MetricsResponse is a response that backend instances use to report their status
type MetricsResponse struct {
	Scheduler scheduler.ScalerMetrics `json:"scheduler"`
}

// watcher watches a "replicaSet" of Sats for a single FQMN
type watcher struct {
	fqmn      string
	instances map[string]*instance
	log       *vlog.Logger
}

type instance struct {
	fqmn    string
	metrics *MetricsResponse
	uuid    string
	pid     int
}

type watcherReport struct {
	instCount    int
	totalThreads int
	failedPorts  []string
}

// newWatcher creates a new watcher instance for the given fqmn
func newWatcher(fqmn string, log *vlog.Logger) *watcher {
	return &watcher{
		fqmn:      fqmn,
		instances: map[string]*instance{},
		log:       log,
	}
}

// add inserts a new instance to the watched pool.
func (w *watcher) add(fqmn, port, uuid string, pid int) {
	w.instances[port] = &instance{
		fqmn: fqmn,
		uuid: uuid,
		pid:  pid,
	}
}

// scaleDown terminates one random instance from the pool
func (w *watcher) scaleDown() error {
	// we use the range to get a semi-random instance
	// and then immediately return so that we only terminate one
	for p := range w.instances {
		w.log.Debug("[watcher.scaleDown] scaling down, terminating instance on port", p, "(", w.instances[p].fqmn, ")")

		return w.terminateInstance(p)
	}

	return nil
}

func (w *watcher) terminate() error {
	var err error
	for p, instance := range w.instances {
		w.log.Debug(fmt.Sprintf("[watcher.terminate] terminating instance on port %s", p))

		err = w.terminateInstance(p)
		if err != nil {
			w.log.Warn("[watcher.terminate] could not terminate instance", instance.fqmn, err.Error())
		}
	}

	return err
}

// terminateInstance terminates the instance from the given port
func (w *watcher) terminateInstance(p string) error {
	inst, ok := w.instances[p]
	if !ok {
		return fmt.Errorf("there isn't an instance on port %s", p)
	}

	if err := syscall.Kill(inst.pid, syscall.SIGTERM); err != nil {
		w.log.Warn("[watcher.terminateInstance]syscall.Kill for pid %d failed, will delete procfile", inst.pid)

		if err := process.Delete(inst.uuid); err != nil {
			return errors.Wrapf(err, "failed to process.Delete for port %s / fqmn %s", p, inst.fqmn)
		}
	}

	delete(w.instances, p)

	w.log.Debug(fmt.Sprintf("[watcher.terminateInstance] successfully terminated instance on port %s (%s)", p, inst.fqmn))

	return nil
}

// report fetches a metrics report from each watched instance and returns a summary
func (w *watcher) report() *watcherReport {
	if len(w.instances) == 0 {
		return nil
	}

	totalThreads := 0
	failedPorts := make([]string, 0)

	for p := range w.instances {
		metrics, err := getReport(p)
		if err != nil {
			w.log.Error(errors.Wrapf(err, "failed to getReport for %s", p))
			failedPorts = append(failedPorts, p)
		} else {
			w.instances[p].metrics = metrics
			totalThreads += metrics.Scheduler.TotalThreadCount
		}
	}

	report := &watcherReport{
		instCount:    len(w.instances) - len(failedPorts),
		totalThreads: totalThreads,
		failedPorts:  failedPorts,
	}

	return report
}

// getReport sends a request on localhost to the given port to fetch metrics
func getReport(port string) (*MetricsResponse, error) {
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:%s/meta/metrics", port), nil)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to Do metrics request")
	}

	defer resp.Body.Close()
	metricsJSON, _ := io.ReadAll(resp.Body)

	metrics := &MetricsResponse{}
	if err := json.Unmarshal(metricsJSON, metrics); err != nil {
		return nil, errors.Wrap(err, "failed to Unmarshal metrics response")
	}

	return metrics, nil
}
