package testutil

import (
	"fmt"
	"time"
)

// AsyncCounter is an async counter
type AsyncCounter struct {
	countChan chan struct{}
}

// NewAsyncCounter creates a new AsyncCounter
func NewAsyncCounter(size int) *AsyncCounter {
	t := &AsyncCounter{
		countChan: make(chan struct{}, size),
	}

	return t
}

// Count increments the counter
func (a *AsyncCounter) Count() {
	go func() {
		a.countChan <- struct{}{}
	}()
}

// Wait waits until the total is reached or the timeout happens
func (a *AsyncCounter) Wait(total, timeoutSeconds int) error {
	count := 0
	timeoutMs := int64(timeoutSeconds * 1000)
	start := time.Now()

	for {
		select {
		case <-a.countChan:
			count++
		default:
			// nothing
		}

		if time.Since(start).Milliseconds() > timeoutMs {
			break
		}
	}

	if count != total {
		return fmt.Errorf("AsyncCoutnter got the incorrect count: %d, expected %d (in %d seconds)", count, total, timeoutSeconds)
	}

	return nil
}
