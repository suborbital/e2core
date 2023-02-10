package main

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/suborbital/e2core/sat/sat"
	"github.com/suborbital/e2core/sat/sat/process"
)

type ProcFileMonitor struct {
	conf     *sat.Config
	stopChan chan struct{}
}

func NewMonitor(log zerolog.Logger, conf *sat.Config) (*ProcFileMonitor, error) {
	// write a file to disk which describes this instance
	info := process.NewInfo(conf.Port, conf.JobType)
	if err := info.Write(conf.ProcUUID); err != nil {
		return nil, errors.Wrap(err, "failed to Write process info")
	}

	log.Debug().Str("procUUID", conf.ProcUUID).Msg("procfile created")

	return &ProcFileMonitor{
		conf:     conf,
		stopChan: make(chan struct{})}, nil
}

func (p *ProcFileMonitor) Start(ctx context.Context) error {
	// continually look for the deletion of our procfile
	for {
		select {
		case <-p.stopChan:
			return nil
		default:
			if ctx.Err() != nil {
				break
			}

			if _, err := process.Find(p.conf.ProcUUID); err != nil {
				return errors.Wrap(err, "proc file deleted")
			}

			time.Sleep(time.Second)
		}
	}
}

func (p *ProcFileMonitor) Stop() {
	p.stopChan <- struct{}{}
}
