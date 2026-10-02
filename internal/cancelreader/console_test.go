package cancelreader

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeConsole is an in-memory console queue.
type fakeConsole struct {
	mu        sync.Mutex
	queue     []consoleRecord
	waits     []time.Duration
	peekErr   error
	discErr   error
	discarded int
}

func (f *fakeConsole) push(recs ...consoleRecord) {
	f.mu.Lock()
	f.queue = append(f.queue, recs...)
	f.mu.Unlock()
}

func (f *fakeConsole) wait(d time.Duration) (bool, error) {
	f.mu.Lock()
	f.waits = append(f.waits, d)
	n := len(f.queue)
	f.mu.Unlock()
	if n > 0 {
		return true, nil
	}
	time.Sleep(time.Millisecond) // a slice, much shorter than d
	return false, nil
}

func (f *fakeConsole) peek() ([]consoleRecord, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]consoleRecord(nil), f.queue...), f.peekErr
}

func (f *fakeConsole) discard(n int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.discErr != nil {
		return f.discErr
	}
	f.queue = f.queue[n:]
	f.discarded += n
	return nil
}

var (
	keyRec   = consoleRecord{key: true}
	otherRec = consoleRecord{}
)

func never() bool { return false }

func TestNonKeyEventsAreDiscardedAndReadKeepsWaiting(t *testing.T) {
	c := &fakeConsole{}
	c.push(otherRec, otherRec, otherRec)
	var canceled atomic.Bool
	done := make(chan error, 1)
	go func() { done <- waitForKey(c, canceled.Load) }()

	time.Sleep(30 * time.Millisecond)
	select {
	case err := <-done:
		t.Fatalf("returned %v with only non-key events queued", err)
	default:
	}
	c.mu.Lock()
	left, gone := len(c.queue), c.discarded
	c.mu.Unlock()
	if left != 0 || gone != 3 {
		t.Errorf("queue = %d left, %d discarded; want 0 and 3", left, gone)
	}

	canceled.Store(true)
	select {
	case err := <-done:
		if !errors.Is(err, ErrCanceled) {
			t.Errorf("err = %v, want ErrCanceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("cancel did not stop the wait promptly")
	}
}

func TestKeyAfterNonKeyEventsIsKeptAndReported(t *testing.T) {
	c := &fakeConsole{}
	c.push(otherRec, otherRec, keyRec, otherRec)
	if err := waitForKey(c, never); err != nil {
		t.Fatal(err)
	}
	// The events before the key are gone; the key and what follows stay.
	if len(c.queue) != 2 || !c.queue[0].key || c.discarded != 2 {
		t.Errorf("queue = %+v, discarded %d; want the key and one event left, 2 discarded", c.queue, c.discarded)
	}
}

func TestKeyAtTheFrontIsNotDiscarded(t *testing.T) {
	c := &fakeConsole{}
	c.push(keyRec)
	if err := waitForKey(c, never); err != nil || c.discarded != 0 || len(c.queue) != 1 {
		t.Errorf("err %v, discarded %d, queue %d; want the key untouched", err, c.discarded, len(c.queue))
	}
}

func TestEmptyQueueWaitsInSlicesNoLongerThanBefore(t *testing.T) {
	c := &fakeConsole{}
	var calls atomic.Int32
	err := waitForKey(c, func() bool { return calls.Add(1) > 5 })
	if !errors.Is(err, ErrCanceled) {
		t.Fatalf("err = %v, want ErrCanceled", err)
	}
	if len(c.waits) < 4 {
		t.Fatalf("only %d waits", len(c.waits))
	}
	for _, d := range c.waits {
		if d <= 0 || d > 50*time.Millisecond {
			t.Errorf("wait slice %v, want 0 < d <= 50ms", d)
		}
	}
}

func TestConsoleErrorsAreReturned(t *testing.T) {
	boom := errors.New("boom")

	c := &fakeConsole{peekErr: boom}
	c.push(otherRec)
	if err := waitForKey(c, never); !errors.Is(err, boom) {
		t.Errorf("peek error: got %v", err)
	}

	c = &fakeConsole{discErr: boom}
	c.push(otherRec, keyRec)
	if err := waitForKey(c, never); !errors.Is(err, boom) {
		t.Errorf("discard error: got %v", err)
	}
}

func TestAlreadyCanceledReturnsWithoutTouchingTheConsole(t *testing.T) {
	c := &fakeConsole{}
	c.push(keyRec)
	if err := waitForKey(c, func() bool { return true }); !errors.Is(err, ErrCanceled) {
		t.Errorf("err = %v", err)
	}
	if len(c.waits) != 0 || len(c.queue) != 1 {
		t.Error("a canceled wait touched the console")
	}
}
