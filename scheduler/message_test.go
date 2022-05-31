package scheduler

import (
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/suborbital/grav/grav"
	"github.com/suborbital/grav/testutil"
)

const msgTypeTester = "reactr.test"
const msgTypeNil = "reactr.testnil"

// to test jobs listening to a Grav message
type msgRunner struct{}

func (m *msgRunner) Run(job Job, ctx *Ctx) (interface{}, error) {
	name := string(job.Bytes())

	reply := grav.NewMsg(msgTypeTester, []byte(fmt.Sprintf("hello, %s", name)))

	return reply, nil
}

func (m *msgRunner) OnChange(change ChangeEvent) error { return nil }

// to test jobs with a nil result
type nilRunner struct{}

func (m *nilRunner) Run(job Job, ctx *Ctx) (interface{}, error) {
	return nil, nil
}

func (m *nilRunner) OnChange(change ChangeEvent) error { return nil }

func TestHandleMessage(t *testing.T) {
	r := New()
	g := grav.New()

	r.Register(msgTypeTester, &msgRunner{})
	r.Listen(g.Connect(), msgTypeTester)

	counter := testutil.NewAsyncCounter(10)

	sender := g.Connect()

	sender.OnType(msgTypeTester, func(msg grav.Message) error {
		counter.Count()
		return nil
	})

	sender.Send(grav.NewMsg(msgTypeTester, []byte("charlie brown")))

	if err := counter.Wait(1, 1); err != nil {
		t.Error(errors.Wrap(err, "failed to counter.Wait"))
	}
}

func TestHandleMessagePt2(t *testing.T) {
	r := New()
	g := grav.New()

	r.Register(msgTypeTester, &msgRunner{})
	r.Listen(g.Connect(), msgTypeTester)

	counter := testutil.NewAsyncCounter(10000)

	sender := g.Connect()

	sender.OnType(msgTypeTester, func(msg grav.Message) error {
		counter.Count()
		return nil
	})

	for i := 0; i < 9876; i++ {
		sender.Send(grav.NewMsg(msgTypeTester, []byte("charlie brown")))
	}

	if err := counter.Wait(9876, 1); err != nil {
		t.Error(errors.Wrap(err, "failed to counter.Wait"))
	}
}

func TestHandleMessageNilResult(t *testing.T) {
	r := New()
	g := grav.New()

	r.Register(msgTypeNil, &nilRunner{})
	r.Listen(g.Connect(), msgTypeNil)

	counter := testutil.NewAsyncCounter(10)

	pod := g.Connect()

	pod.OnType(MsgTypeReactrNilResult, func(msg grav.Message) error {
		counter.Count()
		return nil
	})

	for i := 0; i < 5; i++ {
		pod.Send(grav.NewMsg(msgTypeNil, []byte("hi")))
	}

	if err := counter.Wait(5, 1); err != nil {
		t.Error(errors.Wrap(err, "failed to counter.Wait"))
	}
}
