package grav

import "time"

// TimeoutFunc is a function that takes a value (a number of seconds) and returns a channel that fires after that given amount of time
type TimeoutFunc func() chan time.Time

// Timeout returns a function that returns a channel that fires after the provided number of seconds have elapsed
// if the value passed is less than or equal to 0, the timeout will never fire
func Timeout(seconds int) TimeoutFunc {
	return func() chan time.Time {
		tChan := make(chan time.Time)

		if seconds > 0 {
			go func() {
				duration := time.Second * time.Duration(seconds)
				tChan <- <-time.After(duration)
			}()
		}

		return tChan
	}
}

// TO is a shorthand for Timeout
func TO(seconds int) TimeoutFunc {
	return Timeout(seconds)
}
