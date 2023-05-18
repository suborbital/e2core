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

type Result struct{}
