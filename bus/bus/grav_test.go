package bus

import (
	"fmt"
	"testing"

	"github.com/suborbital/deltav/bus/testutil"
)

func TestGravSingle(t *testing.T) {
	g := New()

	counter := testutil.NewAsyncCounter(5)

	p1 := g.Connect()

	p1.On(func(msg Message) error {
		counter.Count()

		return nil
	})

	p2 := g.Connect()
	p2.Send(NewMsg(MsgTypeDefault, []byte("hello, world")))

	if err := counter.Wait(1, 1); err != nil {
		t.Error(err)
	}
}

func TestGravSanity(t *testing.T) {
	g := New()

	counter := testutil.NewAsyncCounter(100)

	for i := 0; i < 10; i++ {
		p := g.Connect()

		p.On(func(msg Message) error {
			counter.Count()
			return nil
		})
	}

	pod := g.Connect()

	for i := 0; i < 10; i++ {
		pod.Send(NewMsg(MsgTypeDefault, []byte(fmt.Sprintf("hello, world %d", i))))
	}

	if err := counter.Wait(100, 1); err != nil {
		t.Error(err)
	}
}

func TestGravBench(t *testing.T) {
	g := New()

	counter := testutil.NewAsyncCounter(1500000)

	for i := 0; i < 10; i++ {
		p := g.Connect()

		p.On(func(msg Message) error {
			counter.Count()
			return nil
		})
	}

	pod := g.Connect()

	for i := 0; i < 100000; i++ {
		pod.Send(NewMsg(MsgTypeDefault, []byte(fmt.Sprintf("hello, world %d", i))))
	}

	if err := counter.Wait(1000000, 2); err != nil {
		t.Error(err)
	}
}
