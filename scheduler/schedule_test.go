package scheduler

import (
	"testing"

	"github.com/suborbital/grav/testutil"
)

type counterRunner struct {
	counter *testutil.AsyncCounter
}

func (c *counterRunner) Run(job Job, ctx *Ctx) (interface{}, error) {
	c.counter.Count()

	return nil, nil
}

func (c *counterRunner) OnChange(change ChangeEvent) error { return nil }

func TestScheduleAfter(t *testing.T) {
	r := New()

	counter := testutil.NewAsyncCounter(10)

	r.Register("counter", &counterRunner{counter})

	r.Schedule(After(2, func() Job {
		return NewJob("counter", nil)
	}))

	if err := counter.Wait(1, 3); err != nil {
		t.Error(err)
	}
}
