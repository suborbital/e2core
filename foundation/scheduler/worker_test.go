package scheduler

import (
	"log"
	"testing"
	"time"

	"github.com/pkg/errors"
)

func TestReactrJobWithPool(t *testing.T) {
	r := New()

	doGeneric := r.Register("generic", generic{}, PoolSize(3))

	grp := NewGroup()
	grp.Add(doGeneric("first"))
	grp.Add(doGeneric("first"))
	grp.Add(doGeneric("first"))

	if err := grp.Wait(); err != nil {
		log.Fatal(err)
	}
}

type badRunner struct{}

// Run runs a badRunner job
func (g badRunner) Run(job Job, ctx *Ctx) (interface{}, error) {
	return job.String(), nil
}

func (g badRunner) OnChange(change ChangeEvent) error {
	return errors.New("fail")
}

func TestRunnerWithError(t *testing.T) {
	r := New()

	doBad := r.Register("badRunner", badRunner{})

	_, err := doBad(nil).Then()
	if err == nil {
		t.Error("expected error, did not get one")
	}
}

func TestRunnerWithOptionsAndError(t *testing.T) {
	r := New()

	doBad := r.Register("badRunner", badRunner{}, RetrySeconds(1), MaxRetries(1))

	_, err := doBad(nil).Then()
	if err == nil {
		t.Error("expected error, did not get one")
	}
}

type timeoutRunner struct{}

// Run runs a timeoutRunner job
func (g timeoutRunner) Run(job Job, ctx *Ctx) (interface{}, error) {
	time.Sleep(time.Duration(time.Second * 3))

	return nil, nil
}

func (g timeoutRunner) OnChange(change ChangeEvent) error {
	return nil
}

func TestRunnerWithJobTimeout(t *testing.T) {
	r := New()

	doTimeout := r.Register("timeout", timeoutRunner{}, TimeoutSeconds(1))

	if _, err := doTimeout("hello").Then(); err != ErrJobTimeout {
		t.Error("job should have timed out, but did not")
	}
}
