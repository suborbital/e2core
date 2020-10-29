package hive

import (
	"github.com/pkg/errors"
	"github.com/suborbital/grav/grav"
	"github.com/suborbital/vektor/vk"
	"github.com/suborbital/vektor/vlog"
)

const (
	msgTypeHiveJobErr  = "hive.joberr"
	msgTypeHiveTypeErr = "hive.typeerr"
)

// JobFunc is a function that runs a job of a predetermined type
type JobFunc func(interface{}) *Result

// Hive represents the main control object
type Hive struct {
	*scheduler
	log *vlog.Logger
}

// New returns a Hive ready to accept Jobs
func New() *Hive {
	logger := vlog.Default()
	h := &Hive{
		scheduler: newScheduler(logger),
		log:       logger,
	}

	return h
}

// Do schedules a job to be worked on and returns a result object
func (h *Hive) Do(job Job) *Result {
	return h.schedule(job)
}

// Handle registers a Runnable with the Hive and returns a shortcut function to run those jobs
func (h *Hive) Handle(jobType string, runner Runnable, options ...Option) JobFunc {
	h.handle(jobType, runner, options...)

	helper := func(data interface{}) *Result {
		job := NewJob(jobType, data)

		return h.Do(job)
	}

	return helper
}

// HandleMsg registers a Runnable with the Hive and triggers that job whenever the provided Grav pod
// receives a message of a particular type. The message is passed to the runnable as the job data.
// The job's result is then emitted as a message. If the result cannot be cast to type grav.Message,
// or if an error occurs, it is logged and an error is sent. If the result is nil, nothing is sent.
func (h *Hive) HandleMsg(pod *grav.Pod, msgType string, runner Runnable, options ...Option) {
	h.handle(msgType, runner, options...)

	helper := func(data interface{}) *Result {
		job := NewJob(msgType, data)

		return h.Do(job)
	}

	pod.OnType(func(msg grav.Message) error {
		var resultMsg grav.Message

		result, err := helper(msg).Then()
		if err != nil {
			h.log.Error(errors.Wrap(err, "job returned error result"))
			resultMsg = grav.NewMsg(msgTypeHiveJobErr, []byte(err.Error()))
		} else {
			if result == nil {
				return nil
			}

			var ok bool
			resultMsg, ok = result.(grav.Message)
			if !ok {
				h.log.Error(errors.Wrap(err, "job result is not a grav.Message, discarding"))
				resultMsg = grav.NewMsg(msgTypeHiveTypeErr, []byte("failed to convert job result to grav.Message type"))
			}
		}

		pod.Send(resultMsg)

		return nil
	}, msgType)
}

// Job is a shorter alias for NewJob
func (h *Hive) Job(jobType string, data interface{}) Job {
	return NewJob(jobType, data)
}

// Server returns a new Hive server
func (h *Hive) Server(opts ...vk.OptionsModifier) *Server {
	return newServer(h, opts...)
}
