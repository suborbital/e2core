package scheduler

import (
	"fmt"
	"log"
	"testing"

	"github.com/pkg/errors"
	"github.com/suborbital/grav/testutil"
)

type generic struct{}

// Run runs a generic job
func (g generic) Run(job Job, ctx *Ctx) (interface{}, error) {
	if job.String() == "first" {
		return ctx.Do(NewJob("generic", "second")), nil
	} else if job.String() == "second" {
		return ctx.Do(NewJob("generic", "last")), nil
	} else if job.String() == "fail" {
		return nil, errors.New("error")
	}

	return job.String(), nil
}

func (g generic) OnChange(change ChangeEvent) error {
	return nil
}

func TestReactrJob(t *testing.T) {
	r := New()

	r.Register("generic", generic{})

	res := r.Do(r.Job("generic", "first"))

	if res.UUID() == "" {
		t.Error("result ID is empty")
	}

	result, err := res.Then()
	if err != nil {
		log.Fatal(err)
	}

	if result.(string) != "last" {
		t.Error("generic job failed, expected 'last', got", result.(string))
	}
}

type input struct {
	First, Second int
}

type math struct{}

// Run runs a math job
func (g math) Run(job Job, ctx *Ctx) (interface{}, error) {
	in := job.Data().(input)

	return in.First + in.Second, nil
}

func (g math) OnChange(change ChangeEvent) error {
	return nil
}

func TestReactrJobHelperFunc(t *testing.T) {
	r := New()

	doMath := r.Register("math", math{})

	for i := 1; i < 10; i++ {
		answer := i + i*3

		equals, _ := doMath(input{i, i * 3}).ThenInt()
		if equals != answer {
			t.Error("failed to get math right, expected", answer, "got", equals)
		}
	}
}

func TestReactrResultDiscard(t *testing.T) {
	r := New()

	r.Register("generic", generic{})

	res := r.Do(r.Job("generic", "first"))

	// basically just making sure that it doesn't hold up the line
	res.Discard()
}

func TestReactrResultThenDo(t *testing.T) {
	r := New()

	r.Register("generic", generic{})

	wait := make(chan bool)

	r.Do(r.Job("generic", "first")).ThenDo(func(res interface{}, err error) {
		if err != nil {
			t.Error(errors.Wrap(err, "did not expect error"))
			wait <- false
		}

		if res.(string) != "last" {
			t.Error(fmt.Errorf("expected 'last', got %s", res.(string)))
		}

		wait <- true
	})

	r.Do(r.Job("generic", "fail")).ThenDo(func(res interface{}, err error) {
		if err == nil {
			t.Error(errors.New("expected error, did not get one"))
			wait <- false
		}

		wait <- true
	})

	// poor man's async testing
	<-wait
	<-wait
}

type prewarmRunnable struct {
	counter *testutil.AsyncCounter
}

func (p *prewarmRunnable) Run(job Job, ctx *Ctx) (interface{}, error) {
	return nil, nil
}

func (p *prewarmRunnable) OnChange(change ChangeEvent) error {
	if change == ChangeTypeStart {
		p.counter.Count()
	}

	return nil
}

func TestPreWarmWorker(t *testing.T) {
	counter := testutil.NewAsyncCounter(10)

	runnable := &prewarmRunnable{
		counter: counter,
	}

	r := New()
	r.Register("prewarm", runnable, PoolSize(3), PreWarm())

	// checking to see if the prewarmRunnable's OnChange function is called
	// without ever sending it a job (see Runnable above)
	if err := counter.Wait(3, 1); err != nil {
		t.Error(err)
	}
}

func TestDeregisterWorker(t *testing.T) {
	r := New()

	r.Register("generic", generic{})

	res := r.Do(r.Job("generic", "first"))

	if res.UUID() == "" {
		t.Error("result ID is empty")
	}

	result, err := res.Then()
	if err != nil {
		log.Fatal(err)
	}

	if result.(string) != "last" {
		t.Error("generic job failed, expected 'last', got", result.(string))
	}

	if err := r.DeRegister("generic"); err != nil {
		t.Error(errors.Wrap(err, "failed to DeRegister"))
	}

	_, err = r.Do(r.Job("generic", "")).Then()
	if err == nil {
		t.Error("expected error but there was none")
	}
}
