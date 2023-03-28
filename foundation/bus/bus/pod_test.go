package bus

import (
	"bytes"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/suborbital/e2core/foundation/bus/testutil"
)

func TestPodFilter(t *testing.T) {
	g := New()

	counter := testutil.NewAsyncCounter(100)

	onFunc := func(msg Message) error {
		counter.Count()

		return nil
	}

	p1 := g.Connect()
	p1.On(onFunc)

	p2 := g.Connect()
	p2.On(onFunc)

	for i := 0; i < 10; i++ {
		p1.Send(NewMsg(MsgTypeDefault, []byte(fmt.Sprintf("hello, world %d", i))))
	}

	// only 10 should be tracked because p1 should have filtered out the messages that it sent
	// and then they should not reach its own onFunc
	if err := counter.Wait(10, 1); err != nil {
		t.Error(err)
	}
}

func TestPodFilterMessageSentBySelf(t *testing.T) {
	g := New()

	counter := testutil.NewAsyncCounter(100)

	onFunc := func(msg Message) error {
		counter.Count()

		return nil
	}

	p1 := g.Connect()
	p1.On(onFunc)

	p2 := g.Connect()
	//p2.On(onFunc)

	p1.Send(NewMsg(MsgTypeDefault, []byte("hello p2")))
	// Message sent by p1 should end up with self
	if err := counter.Wait(0, 1); err != nil {
		t.Error(err)
	}

	p2.Send(NewMsg(MsgTypeDefault, []byte("hello p1")))
	// Message sent by p2 should end up with p1
	if err := counter.Wait(1, 1); err != nil {
		t.Error(err)
	}
}

func TestWaitOn(t *testing.T) {
	g := New()

	p1 := g.Connect()

	go func() {
		time.Sleep(time.Duration(time.Millisecond * 500))
		p1.Send(NewMsg(MsgTypeDefault, []byte("hello, world")))
		time.Sleep(time.Duration(time.Millisecond * 500))
		p1.Send(NewMsg(MsgTypeDefault, []byte("goodbye, world")))
	}()

	errGoodbye := errors.New("goodbye")

	p2 := g.Connect()

	if err := p2.WaitOn(func(msg Message) error {
		if bytes.Equal(msg.Data(), []byte("hello, world")) {
			return nil
		}

		return ErrMsgNotWanted
	}); err != nil {
		t.Errorf("expected nil error, got %s", err)
	}

	if err := p2.WaitOn(func(msg Message) error {
		if bytes.Equal(msg.Data(), []byte("goodbye, world")) {
			return errGoodbye
		}

		return ErrMsgNotWanted
	}); err != errGoodbye {
		t.Errorf("expected errGoodbye error, got %s", err)
	}
}

const msgTypeBad = "test.bad"

func TestPodFailure(t *testing.T) {
	g := New()

	counter := testutil.NewAsyncCounter(200)

	// create one pod that returns errors on "bad" messages
	p := g.Connect()
	p.On(func(msg Message) error {
		counter.Count()

		if msg.Type() == msgTypeBad {
			return errors.New("bad message")
		}

		return nil
	})

	// and another 9 that don't
	for i := 0; i < 9; i++ {
		p2 := g.Connect()

		p2.On(func(msg Message) error {
			counter.Count()

			return nil
		})
	}

	pod := g.Connect()

	// send 64 "bad" messages (64 reaches the highwater mark)
	for i := 0; i < 64; i++ {
		pod.Send(NewMsg(msgTypeBad, []byte(fmt.Sprintf("hello, world %d", i))))
	}

	time.Sleep(time.Second)

	// send 10 more "bad" messages
	for i := 0; i < 10; i++ {
		pod.Send(NewMsg(msgTypeBad, []byte(fmt.Sprintf("hello, world %d", i))))
	}

	// 730 because the 64th message to the "bad" pod put it over the highwater
	// mark and so the last 10 message would never be delievered
	// sleeps were needed to allow all of the internal goroutines to finish execing
	// in the worst case scenario of a single process machine (which lots of containers are)
	if err := counter.Wait(730, 1); err != nil {
		t.Error(err)
	}

	// the first pod should now have been disconnected, causing only 9 recievers reset and test again

	// send 10 "normal" messages
	for i := 0; i < 10; i++ {
		pod.Send(NewMsg(MsgTypeDefault, []byte(fmt.Sprintf("hello, world %d", i))))
	}

	if err := counter.Wait(90, 1); err != nil {
		t.Error(err)
	}
}

func TestPodFailurePt2(t *testing.T) {
	// test where the "bad" pod is somewhere in the "middle" of the ring
	g := New()

	counter := testutil.NewAsyncCounter(200)

	for i := 0; i < 4; i++ {
		p2 := g.Connect()

		p2.On(func(msg Message) error {
			counter.Count()

			return nil
		})
	}

	// create one pod that returns errors on "bad" messages
	p := g.Connect()
	p.On(func(msg Message) error {
		counter.Count()

		if msg.Type() == msgTypeBad {
			return errors.New("bad message")
		}

		return nil
	})

	// and another 9 that don't
	for i := 0; i < 5; i++ {
		p2 := g.Connect()

		p2.On(func(msg Message) error {
			counter.Count()

			return nil
		})
	}

	pod := g.Connect()

	// send 64 "bad" messages (64 reaches the highwater mark)
	for i := 0; i < 64; i++ {
		pod.Send(NewMsg(msgTypeBad, []byte(fmt.Sprintf("hello, world %d", i))))
	}

	time.Sleep(time.Duration(time.Second))

	// send 10 more "bad" messages
	for i := 0; i < 10; i++ {
		pod.Send(NewMsg(msgTypeBad, []byte(fmt.Sprintf("hello, world %d", i))))
	}

	// 730 because the 64th message to the "bad" pod put it over the highwater
	// mark and so the last 10 message would never be delievered
	// sleeps were needed to allow all of the internal goroutines to finish execing
	// in the worst case scenario of a single process machine (which lots of containers are)
	if err := counter.Wait(730, 1); err != nil {
		t.Error(err)
	}

	// the first pod should now have been disconnected, causing only 9 recievers reset and test again

	// send 10 "normal" messages
	for i := 0; i < 10; i++ {
		pod.Send(NewMsg(MsgTypeDefault, []byte(fmt.Sprintf("hello, world %d", i))))
	}

	if err := counter.Wait(90, 1); err != nil {
		t.Error(err)
	}
}

func TestPodFlushFailed(t *testing.T) {
	g := New()

	counter := testutil.NewAsyncCounter(200)

	// create a pod that returns errors on "bad" messages
	p := g.Connect()
	p.On(func(msg Message) error {
		counter.Count()

		if msg.Type() == msgTypeBad {
			return errors.New("bad message")
		}

		return nil
	})

	sender := g.Connect()

	// send 5 "bad" messages
	for i := 0; i < 5; i++ {
		sender.Send(NewMsg(msgTypeBad, []byte(fmt.Sprintf("hello, world %d", i))))
	}

	<-time.After(time.Duration(time.Second))

	// replace the OnFunc to not error when the flushed messages come back through
	p.On(func(msg Message) error {
		counter.Count()

		return nil
	})

	// send 10 "normal" messages
	for i := 0; i < 9; i++ {
		sender.Send(NewMsg(MsgTypeDefault, []byte(fmt.Sprintf("hello, world %d", i))))
	}

	<-time.After(time.Duration(time.Second))

	// yes this is stupid, but on single-CPU machines (such as GitHub actions), this test won't allow things to be flushed properly.
	sender.Send(NewMsg(MsgTypeDefault, []byte(fmt.Sprintf("flushing!"))))

	// 20 because upon handling the first "good" message, the bus should flush
	// the 5 "failed" messages back into the connection thus repeating them
	if err := counter.Wait(20, 1); err != nil {
		t.Error(err)
	}
}

func TestPodReplay(t *testing.T) {
	g := New()

	counter := testutil.NewAsyncCounter(500)

	// create one pod that returns errors on "bad" messages
	p1 := g.Connect()
	p1.On(func(msg Message) error {
		counter.Count()
		return nil
	})

	sender := g.Connect()

	// send 100 messages and ensure they're received by p1
	for i := 0; i < 100; i++ {
		sender.Send(NewMsg(MsgTypeDefault, []byte(fmt.Sprintf("hello, world %d", i))))
	}

	if err := counter.Wait(100, 1); err != nil {
		t.Error(err)
	}

	// connect a second pod with replay to ensure the same messages come through
	p2 := g.ConnectWithReplay()
	p2.On(func(msg Message) error {
		counter.Count()
		return nil
	})

	sender.Send(NewMsg(MsgTypeDefault, []byte("let's get it started")))

	if err := counter.Wait(102, 1); err != nil {
		t.Error(err)
	}
}

func TestPodReplayPt2(t *testing.T) {
	g := New()

	counter := testutil.NewAsyncCounter(2000)

	p1 := g.Connect()
	p1.On(func(msg Message) error {
		counter.Count()
		return nil
	})

	sender := g.Connect()

	// send 1000 messages and ensure they're received by p1
	for i := 0; i < 1000; i++ {
		sender.Send(NewMsg(MsgTypeDefault, []byte(fmt.Sprintf("hello, world %d", i))))
	}

	if err := counter.Wait(1000, 1); err != nil {
		t.Error(err)
	}

	// connect a second pod with replay to ensure the same messages come through
	p2 := g.ConnectWithReplay()
	p2.On(func(msg Message) error {
		counter.Count()
		return nil
	})

	sender.Send(NewMsg(MsgTypeDefault, []byte(fmt.Sprintf("let's get started"))))

	if err := counter.Wait(130, 1); err != nil {
		t.Error(err)
	}
}

func TestPodDisconnect(t *testing.T) {
	g := New()

	counter := testutil.NewAsyncCounter(10)

	p1 := g.Connect()
	p1.On(func(msg Message) error {
		counter.Count()
		return nil
	})

	p2 := g.Connect()
	p2.On(func(msg Message) error {
		counter.Count()
		return nil
	})

	p3 := g.Connect()

	p1.Disconnect()

	for i := 0; i < 5; i++ {
		p3.Send(NewMsg(MsgTypeDefault, []byte("testing disconnect")))
	}

	// since p1 disconnected, we should only get a count of 5 (from p2, which is still connected)
	if err := counter.Wait(5, 1); err != nil {
		t.Error(err)
	}
}
