package rt

import (
	"sync"

	"github.com/pkg/errors"
)

// ErrJobNotFound and others are storage realated errors
var (
	ErrJobNotFound = errors.New("job not found in storage")
)

// Storage represents a storage driver for Reactr
type Storage interface {
	Add(Job) error
	AddResult(string, interface{}, error) error
	Get(string) (Job, error)
	Remove(string) error
}

// MemoryStorage is the default in-memory storage driver for Reactr
type MemoryStorage struct {
	jobs    sync.Map
	results sync.Map
	errors  sync.Map
}

// a function that can be given to a Result to remove a Job from storage once its result has been delivered
type removeFunc func(string)

func newMemoryStorage() *MemoryStorage {
	m := &MemoryStorage{
		jobs:    sync.Map{},
		results: sync.Map{},
		errors:  sync.Map{},
	}

	return m
}

// Add adds a Job to storage
func (m *MemoryStorage) Add(job Job) error {
	// store as a pointer
	m.jobs.Store(job.UUID(), &job)

	return nil
}

// AddResult adds a Job result to storage
func (m *MemoryStorage) AddResult(uuid string, data interface{}, err error) error {
	if err != nil {
		m.errors.Store(uuid, err.Error())
	} else {
		m.results.Store(uuid, data)
	}

	return nil
}

// Get loads a Job and any of its results from storage
func (m *MemoryStorage) Get(uuid string) (Job, error) {
	rawJob, ok := m.jobs.Load(uuid)
	if !ok {
		return Job{}, ErrJobNotFound
	}

	// cast to pointer as loadResult has a pointer receiver
	job := rawJob.(*Job)

	res, _ := m.results.Load(uuid)

	var errString string

	rawErr, ok := m.errors.Load(uuid)
	if ok {
		errString = rawErr.(string)
	}

	job.loadResult(res, errString)

	return *job, nil
}

// Remove removes a Job and its data from storage
func (m *MemoryStorage) Remove(uuid string) error {
	m.jobs.Delete(uuid)
	m.results.Delete(uuid)
	m.errors.Delete(uuid)

	return nil
}
