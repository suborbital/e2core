package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/suborbital/reactr/rt"
)

var successCount int64 = 0
var failCount int64 = 0

func main() {
	start := time.Now()

	r := rt.New()

	r.Register("loadtest", &loadRunner{}, rt.PoolSize(10))

	group := rt.NewGroup()

	for i := 0; i < 50000; i++ {
		idx := i

		group.Add(
			r.Do(rt.NewJob("loadtest", idx)),
		)
	}

	go func() {
		for {
			req, _ := http.NewRequest(http.MethodGet, "http://localhost:8080/meta/metrics", nil)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				fmt.Println("failed to Do metrics request:", err)
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

func (l *loadRunner) Run(job rt.Job, ctx *rt.Ctx) (interface{}, error) {
	idx := job.Data().(int)

	req, err := http.NewRequest(http.MethodPost, "http://localhost:8080/set/name", strings.NewReader("connor"))
	if err != nil {
		atomic.AddInt64(&failCount, 1)
		return nil, errors.Wrap(err, "failed to NewRequest")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		atomic.AddInt64(&failCount, 1)
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

func (l *loadRunner) OnChange(_ rt.ChangeEvent) error { return nil }
