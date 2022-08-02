package bus

import (
	"fmt"
	"testing"

	"github.com/suborbital/deltav/bus/testutil"
)

func TestRequestReply(t *testing.T) {
	g := New()
	p1 := g.Connect()

	counter := testutil.NewAsyncCounter(10)

	go func() {
		p1.Send(NewMsg(MsgTypeDefault, []byte("joey"))).WaitOn(func(msg Message) error {
			counter.Count()
			return nil
		})
	}()

	p2 := g.ConnectWithReplay()
	p2.On(func(msg Message) error {
		data := string(msg.Data())

		reply := NewMsg(MsgTypeDefault, []byte(fmt.Sprintf("hey %s", data)))
		p2.ReplyTo(msg, reply)

		return nil
	})

	if err := counter.Wait(1, 1); err != nil {
		t.Error(err)
	}
}

func TestRequestReplyAsync(t *testing.T) {
	g := New()
	p1 := g.Connect()

	counter := testutil.NewAsyncCounter(10)

	p1.Send(NewMsg(MsgTypeDefault, []byte("joey"))).OnReply(func(msg Message) error {
		counter.Count()
		return nil
	})

	p2 := g.ConnectWithReplay()
	p2.On(func(msg Message) error {
		data := string(msg.Data())

		reply := NewMsg(MsgTypeDefault, []byte(fmt.Sprintf("hey %s", data)))
		p2.ReplyTo(msg, reply)

		return nil
	})

	if err := counter.Wait(1, 1); err != nil {
		t.Error(err)
	}
}

func TestRequestReplyLoop(t *testing.T) {
	g := New()
	p1 := g.Connect()

	counter := testutil.NewAsyncCounter(2000)

	// testing to ensure calling Send and receipt.Wait in a loop doesn't cause any deadlocks etc.
	go func() {
		for i := 0; i < 1000; i++ {
			p1.Send(NewMsg(MsgTypeDefault, []byte("joey"))).WaitOn(func(msg Message) error {
				counter.Count()
				return nil
			})
		}
	}()

	p2 := g.ConnectWithReplay()
	p2.On(func(msg Message) error {
		data := string(msg.Data())

		reply := NewMsg(MsgTypeDefault, []byte(fmt.Sprintf("hey %s", data)))
		p2.ReplyTo(msg, reply)

		return nil
	})

	if err := counter.Wait(1000, 2); err != nil {
		t.Error(err)
	}
}

func TestRequestReplyTimeout(t *testing.T) {
	g := New()
	p1 := g.Connect()

	counter := testutil.NewAsyncCounter(10)

	go func() {
		if err := p1.Send(NewMsg(MsgTypeDefault, []byte("joey"))).WaitUntil(TO(1), func(msg Message) error {
			return nil
		}); err == ErrWaitTimeout {
			counter.Count()
		}
	}()

	if err := counter.Wait(1, 2); err != nil {
		t.Error(err)
	}
}
