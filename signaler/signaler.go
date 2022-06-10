package signaler

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// Signaler sets up a signal catcing channel and allows
// background tasks to be canceled in unison
type Signaler struct {
	ctx          context.Context
	errChan      chan error
	signalChan   chan os.Signal
	shutdownChan chan struct{}
	group        sync.WaitGroup
}

func Setup() *Signaler {
	shutdownChan := make(chan struct{})
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancelFunc := context.WithCancel(context.Background())

	// if a signal is received, cancel the global context
	// and start the shutdown (handled by Wait())
	go func() {
		<-signalChan
		cancelFunc()
		shutdownChan <- struct{}{}
	}()

	s := &Signaler{
		ctx:          ctx,
		errChan:      make(chan error),
		signalChan:   signalChan,
		shutdownChan: shutdownChan,
		group:        sync.WaitGroup{},
	}

	return s
}

// start starts the given task on a goroutine using the global context
func (s *Signaler) Start(task func(context.Context) error) {
	s.group.Add(1)

	go func() {
		err := task(s.ctx)

		s.group.Done()

		if err != nil {
			s.errChan <- err
		}
	}()
}

// ManualShutdown triggers an artifical shutdown and then calls Wait with the given timeout
func (s *Signaler) ManualShutdown(timeout time.Duration) error {
	s.signalChan <- os.Kill

	return s.Wait(timeout)
}

// Wait blocks until all of the started tasks are completed
// and returns any errors that occur
func (s *Signaler) Wait(timeout time.Duration) error {
	doneChan := make(chan struct{})

	go func() {
		s.group.Wait()
		doneChan <- struct{}{}
	}()

	// fall through when any channel fires
	select {
	case <-doneChan:
	case <-s.shutdownChan:
		select {
		case <-time.After(timeout):
		case <-doneChan:
		}
	case err := <-s.errChan:
		return err
	}

	return nil
}
