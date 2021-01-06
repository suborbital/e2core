package hive

import (
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/suborbital/grav/grav"
	"github.com/suborbital/vektor/vk"
	"github.com/suborbital/vektor/vlog"
)

// MsgTypeHiveJobErr and others are Grav message types used for Hive job
const (
	MsgTypeHiveJobErr    = "hive.joberr"
	MsgTypeHiveResult    = "hive.result"
	MsgTypeHiveNilResult = "hive.nil"
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
	cache := newMemoryCache()

	h := &Hive{
		scheduler: newScheduler(logger, cache),
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
// receives a message of a particular type.
func (h *Hive) HandleMsg(pod *grav.Pod, msgType string, runner Runnable, options ...Option) {
	h.handle(msgType, runner, options...)

	h.Listen(pod, msgType)
}

// Listen causes Hive to listen for messages of the given type and trigger the job of the same type.
// The message's data is passed to the runnable as the job data.
// The job's result is then emitted as a message. If an error occurs, it is logged and an error is sent.
// If the result is nil, nothing is sent.
func (h *Hive) Listen(pod *grav.Pod, msgType string) {
	helper := func(data interface{}) *Result {
		job := NewJob(msgType, data)

		return h.Do(job)
	}

	pod.OnType(msgType, func(msg grav.Message) error {
		var replyMsg grav.Message

		result, err := helper(msg.Data()).Then()
		if err != nil {
			h.log.Error(errors.Wrapf(err, "job from message %s returned error result", msg.UUID()))
			replyMsg = grav.NewMsg(MsgTypeHiveJobErr, []byte(err.Error()))
		} else {
			if result == nil {
				// if the job returned no result
				replyMsg = grav.NewMsg(MsgTypeHiveNilResult, []byte{})
			} else if resultMsg, isMsg := result.(grav.Message); isMsg {
				// if the job returned a Grav message
				resultMsg.SetReplyTo(msg.UUID())
				replyMsg = resultMsg
			} else if bytes, isBytes := result.([]byte); isBytes {
				// if the job returned bytes
				replyMsg = grav.NewMsg(MsgTypeHiveResult, bytes)
			} else if resultString, isString := result.(string); isString {
				// if the job returned a string
				replyMsg = grav.NewMsg(MsgTypeHiveResult, []byte(resultString))
			} else {
				// if the job returned something else like a struct
				resultJSON, err := json.Marshal(result)
				if err != nil {
					h.log.Error(errors.Wrapf(err, "job from message %s returned result that could not be JSON marshalled", msg.UUID()))
					replyMsg = grav.NewMsg(MsgTypeHiveJobErr, []byte(errors.Wrap(err, "failed to Marshal job result").Error()))
				}

				replyMsg = grav.NewMsg(MsgTypeHiveResult, resultJSON)
			}
		}

		pod.ReplyTo(msg, replyMsg)

		return nil
	})
}

// Job is a shorter alias for NewJob
func (h *Hive) Job(jobType string, data interface{}) Job {
	return NewJob(jobType, data)
}

// Server returns a new Hive server
func (h *Hive) Server(opts ...vk.OptionsModifier) *Server {
	return newServer(h, opts...)
}
