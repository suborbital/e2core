package rt

import "time"

/**
This is not to be confused with the `scheduler` type, which is internal to the Reactr instance and actually schedules
jobs for the registered workers. `Schedule` is an external type that allows the caller to define a literal schedule
for jobs to run.
**/

// Schedule is a type that returns an *optional* job if there is something that should be scheduled.
// Reactr will poll the Check() method at regular intervals to see if work is available.
type Schedule interface {
	Check() *Job
	Done() bool
}

type everySchedule struct {
	jobFunc func() Job
	seconds int
	last    *time.Time
}

// Every returns a Schedule that will schedule the job provided by jobFunc every x seconds
func Every(seconds int, jobFunc func() Job) Schedule {
	e := &everySchedule{
		jobFunc: jobFunc,
		seconds: seconds,
	}

	return e
}

func (e *everySchedule) Check() *Job {
	now := time.Now()

	// return a job if this schedule has never been checked OR the 'last' job was more than x seconds ago
	if e.last == nil || time.Since(*e.last).Seconds() >= float64(e.seconds) {
		e.last = &now

		job := e.jobFunc()
		return &job
	}

	return nil
}

func (e *everySchedule) Done() bool {
	return false
}

type afterSchedule struct {
	jobFunc func() Job
	seconds int
	created time.Time
	done    bool
}

// After returns a schedule that will schedule the job provided by jobFunc one time x seconds after creation
func After(seconds int, jobFunc func() Job) Schedule {
	a := &afterSchedule{
		jobFunc: jobFunc,
		seconds: seconds,
		created: time.Now(),
		done:    false,
	}

	return a
}

func (a *afterSchedule) Check() *Job {
	if time.Since(a.created).Seconds() >= float64(a.seconds) {
		a.done = true
		job := a.jobFunc()

		return &job
	}

	return nil
}

func (a *afterSchedule) Done() bool {
	return a.done
}
