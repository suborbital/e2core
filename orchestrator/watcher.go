package orchestrator

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/pkg/errors"

	"github.com/suborbital/reactr/rt"
	"github.com/suborbital/vektor/vlog"

	"github.com/suborbital/velocity/orchestrator/process"
)

var client = http.Client{Timeout: time.Second}

// MetricsResponse is a response that backend instances use to report their status
type MetricsResponse struct {
	Scheduler rt.ScalerMetrics `json:"scheduler"`
}

// watcher watches a "replicaSet" of Sats for a single FQFN
type watcher struct {
	fqfn      string
	instances map[string]*instance
	log       *vlog.Logger
}

type instance struct {
	metrics *MetricsResponse
	uuid    string
	pid     int
}

type watcherReport struct {
	instCount    int
	totalThreads int
	failedPorts  []string
}

// newWatcher creates a new watcher instance for the given fqfn
func newWatcher(fqfn string, log *vlog.Logger) *watcher {
	w := &watcher{
		fqfn:      fqfn,
		instances: map[string]*instance{},
		log:       log,
	}

	return w
}

// add adds a new instance to the watched pool
func (w *watcher) add(port, uuid string, pid int) {
	w.instances[port] = &instance{
		uuid: uuid,
		pid:  pid,
	}
}

// terminate terminates a random instance from the pool
func (w *watcher) terminate() error {
	// we use the range to get a semi-random instance
	// and then immediately return so that we only terminate one
	for p := range w.instances {
		w.log.Info("terminating instance on port", p)

		return w.terminateInstance(p)
	}

	return nil
}

// terminateInstance terminates the instance from the given port
func (w *watcher) terminateInstance(p string) error {
	inst := w.instances[p]
	delete(w.instances, p)

	if inst != nil {
		if err := process.Delete(inst.uuid); err != nil {
			return errors.Wrap(err, "failed to process.Delete")
		}
	}

	return nil
}

// report fetches a metrics report from each watched instance and returns a summary
func (w *watcher) report() *watcherReport {
	if len(w.instances) == 0 {
		return nil
	}

	totalThreads := 0
	failedPorts := []string{}

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

	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to Do metrics request")
	}

	defer resp.Body.Close()
	metricsJSON, _ := ioutil.ReadAll(resp.Body)

	metrics := &MetricsResponse{}
	if err := json.Unmarshal(metricsJSON, metrics); err != nil {
		return nil, errors.Wrap(err, "failed to Unmarshal metrics response")
	}

	return metrics, nil
}
