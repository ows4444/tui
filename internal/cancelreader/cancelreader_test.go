//go:build darwin || linux

package cancelreader

import (
	"errors"
	"io"
	"os"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/ows4444/tui/internal/ptytest"
)

type readResult struct {
	b   []byte
	err error
}

// readAsync starts one Read on r in the background.
func readAsync(r *Reader, size int) <-chan readResult {
	ch := make(chan readResult, 1)
	go func() {
		buf := make([]byte, size)
		n, err := r.Read(buf)
		ch <- readResult{buf[:n], err}
	}()
	return ch
}

func expect(t *testing.T, ch <-chan readResult, d time.Duration) readResult {
	t.Helper()
	select {
	case r := <-ch:
		return r
	case <-time.After(d):
		t.Fatal("Read did not return in time")
		return readResult{}
	}
}

func newPipe(t *testing.T) (*Reader, *os.File, *os.File) {
	t.Helper()
	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	r, err := New(pr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		r.Close()
		pr.Close()
		pw.Close()
	})
	return r, pr, pw
}

func TestReadDeliversBytesInOrder(t *testing.T) {
	r, _, pw := newPipe(t)

	pw.WriteString("hello")
	if got := expect(t, readAsync(r, 64), 2*time.Second); string(got.b) != "hello" || got.err != nil {
		t.Errorf("Read = %q, %v; want %q, nil", got.b, got.err, "hello")
	}

	// A small buffer takes the bytes in order across reads, none lost.
	pw.WriteString("abcdef")
	var all []byte
	for len(all) < 6 {
		got := expect(t, readAsync(r, 4), 2*time.Second)
		if got.err != nil {
			t.Fatalf("Read: %v", got.err)
		}
		all = append(all, got.b...)
	}
	if string(all) != "abcdef" {
		t.Errorf("read %q, want %q", all, "abcdef")
	}
}

func TestReadBlocksUntilInputArrives(t *testing.T) {
	r, _, pw := newPipe(t)
	ch := readAsync(r, 8)
	select {
	case got := <-ch:
		t.Fatalf("Read returned early with %q, %v", got.b, got.err)
	case <-time.After(100 * time.Millisecond):
	}
	pw.WriteString("x")
	if got := expect(t, ch, 2*time.Second); string(got.b) != "x" || got.err != nil {
		t.Errorf("Read = %q, %v; want x", got.b, got.err)
	}
}

func TestReadReportsEOF(t *testing.T) {
	r, _, pw := newPipe(t)
	pw.WriteString("z")
	pw.Close()
	if got := expect(t, readAsync(r, 8), 2*time.Second); string(got.b) != "z" {
		t.Errorf("first Read = %q, %v; want the pending byte", got.b, got.err)
	}
	if got := expect(t, readAsync(r, 8), 2*time.Second); !errors.Is(got.err, io.EOF) {
		t.Errorf("second Read err = %v, want io.EOF", got.err)
	}
}

func TestCancelUnblocksAWaitingRead(t *testing.T) {
	r, _, _ := newPipe(t)
	ch := readAsync(r, 8)
	time.Sleep(50 * time.Millisecond) // let it block
	start := time.Now()
	r.Cancel()
	got := expect(t, ch, time.Second)
	if !errors.Is(got.err, ErrCanceled) || len(got.b) != 0 {
		t.Errorf("Read = %q, %v; want no bytes and ErrCanceled", got.b, got.err)
	}
	if d := time.Since(start); d > 500*time.Millisecond {
		t.Errorf("Cancel took %v to unblock Read", d)
	}
}

func TestReadAfterCancelStaysCanceled(t *testing.T) {
	r, _, pw := newPipe(t)
	r.Cancel()
	pw.WriteString("late")
	for i := 0; i < 3; i++ {
		if got := expect(t, readAsync(r, 8), time.Second); !errors.Is(got.err, ErrCanceled) || len(got.b) != 0 {
			t.Errorf("Read #%d after Cancel = %q, %v; want ErrCanceled", i, got.b, got.err)
		}
	}
}

func TestCancelIsIdempotentAndConcurrencySafe(t *testing.T) {
	r, _, _ := newPipe(t)
	ch := readAsync(r, 8)
	time.Sleep(20 * time.Millisecond)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				r.Cancel()
			}
		}()
	}
	wg.Wait()
	if got := expect(t, ch, time.Second); !errors.Is(got.err, ErrCanceled) {
		t.Errorf("err = %v, want ErrCanceled", got.err)
	}
	r.Close()
	r.Close() // idempotent
}

func TestCancelDoesNotConsumeInput(t *testing.T) {
	r, pr, pw := newPipe(t)
	pw.WriteString("keep")
	r.Cancel()
	if got := expect(t, readAsync(r, 8), time.Second); !errors.Is(got.err, ErrCanceled) {
		t.Fatalf("err = %v, want ErrCanceled", got.err)
	}
	// The bytes are still there for the next reader of the file.
	buf := make([]byte, 8)
	n, err := pr.Read(buf)
	if err != nil || string(buf[:n]) != "keep" {
		t.Errorf("next reader got %q, %v; want %q", buf[:n], err, "keep")
	}
}

// The case that motivated this package: a terminal that Program.Run has put
// in blocking mode with File.Fd(), where SetReadDeadline no longer
// interrupts a Read. Cancel must still free the reader, and keys typed
// afterwards must go to the next reader rather than a stale one.
func TestCancelWorksOnABlockingTerminal(t *testing.T) {
	master, slave, err := ptytest.Open()
	if err != nil {
		t.Skipf("cannot open a pty here: %v", err)
	}
	defer master.Close()
	defer slave.Close()
	_ = slave.Fd() // what Program.Run does; puts the file in blocking mode

	r, err := New(slave)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	// Confirm the premise: the deadline really is ineffective here.
	slave.SetReadDeadline(time.Now())
	ch := readAsync(r, 8)
	time.Sleep(50 * time.Millisecond)
	r.Cancel()
	got := expect(t, ch, time.Second)
	if !errors.Is(got.err, ErrCanceled) {
		t.Fatalf("Read err = %v, want ErrCanceled", got.err)
	}

	// A second reader on the same terminal gets the next keys, in raw mode
	// so no newline is needed.
	if _, err := ptyRaw(slave); err != nil {
		t.Skipf("cannot set raw mode: %v", err)
	}
	r2, err := New(slave)
	if err != nil {
		t.Fatal(err)
	}
	defer r2.Close()
	ch2 := readAsync(r2, 8)
	time.Sleep(30 * time.Millisecond)
	syscall.Write(int(mustFd(master)), []byte("k"))
	if got := expect(t, ch2, 2*time.Second); string(got.b) != "k" || got.err != nil {
		t.Errorf("second reader got %q, %v; want %q", got.b, got.err, "k")
	}
}

func TestCloseReleasesTheCancelPipeAndCancelAfterCloseIsSafe(t *testing.T) {
	r, _, _ := newPipe(t)
	r.Close()
	// Both ends of the cancel pipe are closed: closing them again reports it.
	if err := r.cancelR.Close(); !errors.Is(err, os.ErrClosed) {
		t.Errorf("cancel pipe read end still open after Close: %v", err)
	}
	if err := r.cancelW.Close(); !errors.Is(err, os.ErrClosed) {
		t.Errorf("cancel pipe write end still open after Close: %v", err)
	}
	r.Cancel() // must not write to a closed pipe or panic
	r.Close()
}
