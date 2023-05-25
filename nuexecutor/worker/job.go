package worker

import (
	"context"
)

type Job struct {
	ctx          context.Context
	input        []byte
	errChan      chan error
	responseChan chan Result
}

func (j Job) Input() []byte {
	return j.input
}

func (j Job) Error() <-chan error {
	return j.errChan
}

func (j Job) Result() <-chan Result {
	return j.responseChan
}

type Result struct {
	content []byte
}

func (r Result) Output() []byte {
	return r.content
}

func NewJob(ctx context.Context, payload []byte) Job {
	return Job{
		ctx:          ctx,
		input:        payload,
		errChan:      make(chan error),
		responseChan: make(chan Result),
	}
}
