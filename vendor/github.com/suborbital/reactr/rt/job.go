package rt

import (
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/suborbital/reactr/request"
)

// Job describes a job to be done
type Job struct {
	uuid    string
	jobType string
	result  *Result
	data    interface{}

	caps *Capabilities
	req  *request.CoordinatedRequest
}

// NewJob creates a new job
func NewJob(jobType string, data interface{}) Job {
	j := Job{
		uuid:    uuid.New().String(),
		jobType: jobType,
		data:    data,
	}

	// detect the coordinated request
	if req, ok := data.(*request.CoordinatedRequest); ok {
		j.req = req
	}

	return j
}

func (j Job) UUID() string {
	return j.uuid
}

// Unmarshal unmarshals the job's data into a struct
func (j Job) Unmarshal(target interface{}) error {
	if bytes, ok := j.data.([]byte); ok {
		return json.Unmarshal(bytes, target)
	}

	return errors.New("failed to Unmarshal, job data is not []byte")
}

// String returns the string value of a job's data
func (j Job) String() string {
	if s, isString := j.data.(string); isString {
		return s
	} else if b, isBytes := j.data.([]byte); isBytes {
		return string(b)
	}

	return ""
}

// Bytes returns the []byte value of the job's data
func (j Job) Bytes() []byte {
	if v, ok := j.data.([]byte); ok {
		return v
	} else if s, ok := j.data.(string); ok {
		return []byte(s)
	}

	return nil
}

// Int returns the int value of the job's data
func (j Job) Int() int {
	if v, ok := j.data.(int); ok {
		return v
	}

	return 0
}

// Data returns the "raw" data for the job
func (j Job) Data() interface{} {
	return j.data
}

// Req returns the Coordinated request attached to the Job
func (j Job) Req() *request.CoordinatedRequest {
	return j.req
}
