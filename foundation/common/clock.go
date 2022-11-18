package common

import "time"

type Clock interface {
	// Now returns the current clock time.
	Now() time.Time
	// In returns the clock time plus time.Duration
	In(time.Duration) time.Time
}

// SystemTime is a clock facade on time.Time.
func SystemTime() Clock {
	return &systemClock{}
}

type systemClock struct{}

func (clock *systemClock) Now() time.Time {
	return time.Now()
}

func (clock *systemClock) In(duration time.Duration) time.Time {
	return time.Now().Add(duration)
}

type StableClock interface {
	Clock
	// Tick increments StableTime by duration
	Tick(duration time.Duration)
}

// StableTime is a clock which must be manually updated.
func StableTime(epoch time.Time) StableClock {
	return &stableClock{
		epoch: NewAtomicReference(epoch),
	}
}

type stableClock struct {
	epoch *AtomicReference[time.Time]
}

func (clock *stableClock) Now() time.Time {
	return clock.epoch.Load()
}

func (clock *stableClock) In(duration time.Duration) time.Time {
	return clock.epoch.Load().Add(duration)
}

func (clock *stableClock) Tick(duration time.Duration) {
	clock.epoch.Swap(clock.epoch.Load().Add(duration))
}
