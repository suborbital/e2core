package scheduler

import (
	"encoding/json"
	"os"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/suborbital/e2core/foundation/bus/bus"
)

// MsgTypeReactrJobErr and others are Grav message types used for Scheduler job
const (
	MsgTypeReactrJobErr    = "reactr.joberr" // any kind of error from a job run
	MsgTypeReactrRunErr    = "reactr.runerr" // specifically a RunErr returned from a Wasm Runnable
	MsgTypeReactrResult    = "reactr.result"
	MsgTypeReactrNilResult = "reactr.nil"
)

// JobFunc is a function that runs a job of a predetermined type
type JobFunc func(interface{}) *Result

// Scheduler represents the main control object
type Scheduler struct {
	log  zerolog.Logger
	core *core
}

// New returns a Scheduler ready to accept Jobs
func New() *Scheduler {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	return NewWithLogger(zerolog.New(os.Stderr).With().
		Timestamp().
		Str("component", "scheduler").
		Logger())
}

// NewWithLogger returns a Scheduler with a custom logger
func NewWithLogger(log zerolog.Logger) *Scheduler {
	c := newCore(log)

	r := &Scheduler{
		core: c,
		log:  log,
	}

	return r
}

// Do schedules a job to be worked on and returns a result object
func (r *Scheduler) Do(job Job) *Result {
	return r.core.do(&job)
}

// Schedule adds a new Schedule to the instance, Scheduler will 'watch' the Schedule
// and Do any jobs when the Schedule indicates it's needed
func (r *Scheduler) Schedule(s Schedule) {
	r.core.watch(s)
}

// Register registers a Runnable with the Scheduler and returns a shortcut function to run those jobs
func (r *Scheduler) Register(jobType string, runner Runnable, options ...Option) JobFunc {
	r.core.register(jobType, runner, options...)

	helper := func(data interface{}) *Result {
		return r.Do(NewJob(jobType, data))
	}

	return helper
}

// DeRegister stops the workers for a given jobType and removes it
func (r *Scheduler) DeRegister(jobType string) error {
	return r.core.deRegister(jobType)
}

// Listen causes Scheduler to listen for messages of the given type and trigger the job of the same type.
// The message's data is passed to the runnable as the job data.
// The job's result is then emitted as a message. If an error occurs, it is logged and an error is sent.
// If the result is nil, nothing is sent.
func (r *Scheduler) Listen(pod *bus.Pod, msgType string) {
	r.ListenAndRun(pod, msgType, func(msg bus.Message, result interface{}, err error) {
		var replyMsg bus.Message

		if err != nil {
			r.log.Err(err).Str("messageUUID", msg.UUID()).Msg("job returned error result")

			runErr := &RunErr{}
			if errors.As(err, runErr) {
				// if a Wasm Runnable returned a RunErr, let's be sure to handle that
				replyMsg = bus.NewMsgWithParentID(MsgTypeReactrRunErr, msg.ParentID(), []byte(runErr.Error()))
			} else {
				replyMsg = bus.NewMsgWithParentID(MsgTypeReactrJobErr, msg.ParentID(), []byte(err.Error()))
			}
		} else {
			if result == nil {
				// if the job returned no result
				replyMsg = bus.NewMsgWithParentID(MsgTypeReactrNilResult, msg.ParentID(), []byte{})
			} else if resultMsg, isMsg := result.(bus.Message); isMsg {
				// if the job returned a Grav message
				resultMsg.SetReplyTo(msg.UUID())
				replyMsg = resultMsg
			} else if bytes, isBytes := result.([]byte); isBytes {
				// if the job returned bytes
				replyMsg = bus.NewMsgWithParentID(MsgTypeReactrResult, msg.ParentID(), bytes)
			} else if resultString, isString := result.(string); isString {
				// if the job returned a string
				replyMsg = bus.NewMsgWithParentID(MsgTypeReactrResult, msg.ParentID(), []byte(resultString))
			} else {
				// if the job returned something else like a struct
				resultJSON, err := json.Marshal(result)
				if err != nil {
					r.log.Err(err).Str("messageUUID", msg.UUID()).Msg("job result could not be marshaled into JSON")
					replyMsg = bus.NewMsgWithParentID(MsgTypeReactrJobErr, msg.ParentID(), []byte(errors.Wrap(err, "failed to Marshal job result").Error()))
				} else {
					replyMsg = bus.NewMsgWithParentID(MsgTypeReactrResult, msg.ParentID(), resultJSON)
				}
			}
		}

		pod.ReplyTo(msg, replyMsg)
	})
}

// ListenAndRun subscribes Scheduler to a messageType and calls `run` for each job result
func (r *Scheduler) ListenAndRun(pod *bus.Pod, msgType string, run func(bus.Message, interface{}, error)) {
	helper := func(data interface{}) *Result {
		job := NewJob(msgType, data)

		return r.Do(job)
	}

	// each time a message is received with the associated type,
	// execute the associated job and pass the result to `run`
	pod.OnType(msgType, func(msg bus.Message) error {
		result, err := helper(msg.Data()).Then()

		run(msg, result, err)

		return nil
	})
}

// IsRegistered returns true if the instance
// has a worker registered for the given jobType
func (r *Scheduler) IsRegistered(jobType string) bool {
	return r.core.hasWorker(jobType)
}

// Job is a shorter alias for NewJob
func (r *Scheduler) Job(jobType string, data interface{}) Job {
	return NewJob(jobType, data)
}

// Metrics returns a snapshot in time describing Scheduler's internals
func (r *Scheduler) Metrics() ScalerMetrics {
	return r.core.metrics()
}
