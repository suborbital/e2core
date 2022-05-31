package scheduler

import "testing"

func TestCreateJob(t *testing.T) {
	job := NewJob("test", []byte("{\"some\": 1}"))

	if string(job.data.([]byte)) != "{\"some\": 1}" {
		t.Error("job data incorrect, expected '{\"some\": 1}', got", job.data)
	}

	if job.jobType != "test" {
		t.Error("job type incorrect, expected 'test', got", job.jobType)
	}

	if job.result != nil {
		t.Error("job's result should be empty, is not")
	}
}
