package rt

import (
	"sync/atomic"
	"time"
)

// rateTracker counts a set of events over time and then returns
// the average number of events per second
type rateTracker struct {
	count int64
	last  time.Time
}

func newRateTracker() *rateTracker {
	r := &rateTracker{
		count: 0,
		last:  time.Now(),
	}

	return r
}

func (r *rateTracker) add() {
	atomic.AddInt64(&r.count, 1)
}

func (r *rateTracker) average() float64 {
	seconds := time.Since(r.last).Seconds()

	val := atomic.SwapInt64(&r.count, 0)
	avg := float64(val) / seconds

	r.last = time.Now()

	return avg
}
