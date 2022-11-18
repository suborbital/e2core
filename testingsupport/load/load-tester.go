package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"

	"github.com/suborbital/e2core/foundation/scheduler"
)

var successCount int64 = 0
var failCount int64 = 0

var client = http.Client{Timeout: time.Duration(time.Second * 3)}

func main() {
	start := time.Now()

	r := scheduler.New()

	r.Register("loadtest", &loadRunner{}, scheduler.PoolSize(15))

	group := scheduler.NewGroup()

	for i := 0; i < 50000; i++ {
		idx := i

		group.Add(
			r.Do(scheduler.NewJob("loadtest", idx)),
		)
	}

	go func() {
		for {
			req, _ := http.NewRequest(http.MethodGet, "http://local.suborbital.network:8080/meta/metrics", nil)

			resp, err := client.Do(req)
			if err != nil {
				fmt.Println("failed to Do metrics request:", err)
				continue
			}

			metricsJSON, _ := ioutil.ReadAll(resp.Body)
			resp.Body.Close()

			fmt.Println(string(metricsJSON))

			time.Sleep(time.Second)
		}
	}()

	if err := group.Wait(); err != nil {
		fmt.Println("errors encountered!", err)
	}

	fmt.Printf("success: %d, fail: %d\n", successCount, failCount)
	fmt.Printf("completed in %f s\n", time.Since(start).Seconds())
}

type loadRunner struct{}

func (l *loadRunner) Run(job scheduler.Job, ctx *scheduler.Ctx) (interface{}, error) {
	idx := job.Data().(int)

	req, err := http.NewRequest(http.MethodPost, "http://157.230.68.218/com.suborbital.acmeco/default/httpget/v1.0.0", strings.NewReader("connor"))
	if err != nil {
		atomic.AddInt64(&failCount, 1)
		return nil, errors.Wrap(err, "failed to NewRequest")
	}

	resp, err := client.Do(req)
	if err != nil {
		atomic.AddInt64(&failCount, 1)
		fmt.Println(err)
		return nil, errors.Wrap(err, "failed to to Do request")
	}

	if resp.StatusCode != 200 {
		atomic.AddInt64(&failCount, 1)
		err := fmt.Errorf("non-200 status code %d for idx %d", resp.StatusCode, idx)
		fmt.Println(err)
		return nil, err
	}

	atomic.AddInt64(&successCount, 1)

	return nil, nil
}

func (l *loadRunner) OnChange(_ scheduler.ChangeEvent) error { return nil }
