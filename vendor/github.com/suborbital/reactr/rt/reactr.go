package rt

import (
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/suborbital/grav/grav"
	"github.com/suborbital/vektor/vlog"
)

// MsgTypeReactrJobErr and others are Grav message types used for Reactr job
const (
	MsgTypeReactrJobErr    = "reactr.joberr" // any kind of error from a job run
	MsgTypeReactrRunErr    = "reactr.runerr" // specifically a RunErr returned from a Wasm Runnable
	MsgTypeReactrResult    = "reactr.result"
	MsgTypeReactrNilResult = "reactr.nil"
)

// JobFunc is a function that runs a job of a predetermined type
type JobFunc func(interface{}) *Result

// Reactr represents the main control object
type Reactr struct {
	scheduler *scheduler
	log       *vlog.Logger
}

// New returns a Reactr ready to accept Jobs
func New() *Reactr {
	logger := vlog.Default()
	cache := newMemoryCache()

	h := &Reactr{
		scheduler: newScheduler(logger, cache),
		log:       logger,
	}

	return h
}

// Do schedules a job to be worked on and returns a result object
func (h *Reactr) Do(job Job) *Result {
	return h.scheduler.schedule(job)
}

// Schedule adds a new Schedule to the instance, Reactr will 'watch' the Schedule
// and Do any jobs when the Schedule indicates it's needed
func (h *Reactr) Schedule(s Schedule) {
	h.scheduler.watch(s)
}

// Handle registers a Runnable with the Reactr and returns a shortcut function to run those jobs
func (h *Reactr) Handle(jobType string, runner Runnable, options ...Option) JobFunc {
	h.scheduler.handle(jobType, runner, options...)

	helper := func(data interface{}) *Result {
		job := NewJob(jobType, data)

		return h.Do(job)
	}

	return helper
}

// HandleMsg registers a Runnable with the Reactr and triggers that job whenever the provided Grav pod
// receives a message of a particular type.
func (h *Reactr) HandleMsg(pod *grav.Pod, msgType string, runner Runnable, options ...Option) {
	h.scheduler.handle(msgType, runner, options...)

	h.Listen(pod, msgType)
}

// Listen causes Reactr to listen for messages of the given type and trigger the job of the same type.
// The message's data is passed to the runnable as the job data.
// The job's result is then emitted as a message. If an error occurs, it is logged and an error is sent.
// If the result is nil, nothing is sent.
func (h *Reactr) Listen(pod *grav.Pod, msgType string) {
	helper := func(data interface{}) *Result {
		job := NewJob(msgType, data)

		return h.Do(job)
	}

	pod.OnType(msgType, func(msg grav.Message) error {
		var replyMsg grav.Message

		result, err := helper(msg.Data()).Then()
		if err != nil {
			h.log.Error(errors.Wrapf(err, "job from message %s returned error result", msg.UUID()))

			runErr := &RunErr{}
			if errors.As(err, runErr) {
				// if a Wasm Runnable returned a RunErr, let's be sure to handle that
				replyMsg = grav.NewMsg(MsgTypeReactrRunErr, []byte(runErr.Error()))
			} else {
				replyMsg = grav.NewMsg(MsgTypeReactrJobErr, []byte(err.Error()))
			}
		} else {
			if result == nil {
				// if the job returned no result
				replyMsg = grav.NewMsg(MsgTypeReactrNilResult, []byte{})
			} else if resultMsg, isMsg := result.(grav.Message); isMsg {
				// if the job returned a Grav message
				resultMsg.SetReplyTo(msg.UUID())
				replyMsg = resultMsg
			} else if bytes, isBytes := result.([]byte); isBytes {
				// if the job returned bytes
				replyMsg = grav.NewMsg(MsgTypeReactrResult, bytes)
			} else if resultString, isString := result.(string); isString {
				// if the job returned a string
				replyMsg = grav.NewMsg(MsgTypeReactrResult, []byte(resultString))
			} else {
				// if the job returned something else like a struct
				resultJSON, err := json.Marshal(result)
				if err != nil {
					h.log.Error(errors.Wrapf(err, "job from message %s returned result that could not be JSON marshalled", msg.UUID()))
					replyMsg = grav.NewMsg(MsgTypeReactrJobErr, []byte(errors.Wrap(err, "failed to Marshal job result").Error()))
				}

				replyMsg = grav.NewMsg(MsgTypeReactrResult, resultJSON)
			}
		}

		pod.ReplyTo(msg, replyMsg)

		return nil
	})
}

// Job is a shorter alias for NewJob
func (h *Reactr) Job(jobType string, data interface{}) Job {
	return NewJob(jobType, data)
}
