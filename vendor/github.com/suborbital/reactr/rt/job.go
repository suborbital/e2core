package rt

import (
	"encoding/json"
	"errors"

	"github.com/google/uuid"
)

// JobReference is a lightweight reference to a Job
type JobReference struct {
	uuid    string
	jobType string
	result  *Result
}

// Job describes a job to be done
type Job struct {
	JobReference
	data       interface{}
	resultData interface{}
	resultErr  error
}

// NewJob creates a new job
func NewJob(jobType string, data interface{}) Job {
	j := Job{
		JobReference: JobReference{
			uuid:    uuid.New().String(),
			jobType: jobType,
		},
		data: data,
	}

	return j
}

// UUID returns the Job's UUID
func (j JobReference) UUID() string {
	return j.uuid
}

// Reference returns a reference to the Job
func (j Job) Reference() JobReference {
	return j.JobReference
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

// loadResult has a pointer reciever such that it actually modifies the object it's being called on
func (j *Job) loadResult(resultData interface{}, errString string) {
	j.resultData = resultData
	j.resultErr = errors.New(errString)
}
