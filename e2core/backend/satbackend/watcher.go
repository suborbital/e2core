package satbackend

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/suborbital/e2core/foundation/scheduler"
)

var (
	httpClient      = http.Client{Timeout: time.Second}
	errScaleDown    = errors.New("scaling down watcher by removing a random instance")
	errTerminateAll = errors.New("terminating all instances in watcher")
	errTerminateOne = errors.New("terminating this specific instance")
)

// MetricsResponse is a response that backend instances use to report their status
type MetricsResponse struct {
	Scheduler scheduler.ScalerMetrics `json:"scheduler"`
}

// watcher watches a "replicaSet" of Sats for a single FQMN
type watcher struct {
	fqmn      string
	instances map[string]*instance
	log       zerolog.Logger
}

type instance struct {
	fqmn    string
	metrics *MetricsResponse
	uuid    string
	cxl     context.CancelCauseFunc
}

type watcherReport struct {
	instCount    int
	totalThreads int
	failedPorts  []string
}

// newWatcher creates a new watcher instance for the given fqmn
func newWatcher(fqmn string, log zerolog.Logger) *watcher {
	return &watcher{
		fqmn:      fqmn,
		instances: map[string]*instance{},
		log:       log.With().Str("module", "watcher").Logger(),
	}
}

// add inserts a new instance to the watched pool.
func (w *watcher) add(fqmn, port, uuid string, cxl context.CancelCauseFunc) {
	w.instances[port] = &instance{
		fqmn: fqmn,
		uuid: uuid,
		cxl:  cxl,
	}
}

// scaleDown terminates one random instance from the pool
func (w *watcher) scaleDown() {
	ll := w.log.With().Str("module", "scaleDown").Logger()
	// we use the range to get a semi-random instance
	// and then immediately return so that we only terminate one
	for p := range w.instances {
		ll.Debug().Str("fqmn", w.instances[p].fqmn).Str("port", p).Msg("scaling down, terminating instance")

		w.instances[p].cxl(errScaleDown)

		break
	}
}

func (w *watcher) terminate() {
	ll := w.log.With().Str("method", "terminate").Logger()
	for p, instance := range w.instances {
		ll.Debug().Str("port", p).Msg("terminating instance")

		instance.cxl(errTerminateAll)

		delete(w.instances, p)
	}
}

// terminateInstance terminates the instance from the given port
func (w *watcher) terminateInstance(p string) error {
	inst, ok := w.instances[p]
	if !ok {
		return fmt.Errorf("there isn't an instance on port %s", p)
	}

	inst.cxl(errTerminateOne)

	delete(w.instances, p)

	return nil
}

// report fetches a metrics report from each watched instance and returns a summary
func (w *watcher) report() *watcherReport {
	if len(w.instances) == 0 {
		return nil
	}

	ll := w.log.With().Str("method", "report").Logger()

	totalThreads := 0
	failedPorts := make([]string, 0)

	for p := range w.instances {
		metrics, err := getReport(p)
		if err != nil {
			ll.Err(err).Str("port", p).Msg("getReport failed")
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
