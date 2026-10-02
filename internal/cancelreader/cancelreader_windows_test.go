//go:build windows

package cancelreader

import (
	"errors"
	"io"
	"os"
	"testing"
	"time"
)

func winPipe(t *testing.T) (*Reader, *os.File, *os.File) {
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

type winResult struct {
	b   []byte
	err error
}

func winReadAsync(r *Reader, size int) <-chan winResult {
	ch := make(chan winResult, 1)
	go func() {
		buf := make([]byte, size)
		n, err := r.Read(buf)
		ch <- winResult{buf[:n], err}
	}()
	return ch
}

func winExpect(t *testing.T, ch <-chan winResult) winResult {
	t.Helper()
	select {
	case r := <-ch:
		return r
	case <-time.After(3 * time.Second):
		t.Fatal("Read did not return in time")
		return winResult{}
	}
}

func TestWindowsReadDeliversBytes(t *testing.T) {
	r, _, pw := winPipe(t)
	if _, err := pw.Write([]byte("abc")); err != nil {
		t.Fatal(err)
	}
	if got := winExpect(t, winReadAsync(r, 8)); got.err != nil || string(got.b) != "abc" {
		t.Fatalf("Read = %q, %v; want abc", got.b, got.err)
	}
}

func TestWindowsReadBlocksUntilInputArrives(t *testing.T) {
	r, _, pw := winPipe(t)
	ch := winReadAsync(r, 8)
	select {
	case got := <-ch:
		t.Fatalf("Read returned early: %q, %v", got.b, got.err)
	case <-time.After(100 * time.Millisecond):
	}
	if _, err := pw.Write([]byte("x")); err != nil {
		t.Fatal(err)
	}
	if got := winExpect(t, ch); got.err != nil || string(got.b) != "x" {
		t.Fatalf("Read = %q, %v; want x", got.b, got.err)
	}
}

func TestWindowsReadReportsEOF(t *testing.T) {
	r, _, pw := winPipe(t)
	pw.Close()
	if got := winExpect(t, winReadAsync(r, 8)); !errors.Is(got.err, io.EOF) {
		t.Fatalf("Read error = %v, want io.EOF", got.err)
	}
}

// The failure this Reader exists to fix: a Read parked on a pipe that never
// receives input must return once cancelled.
func TestWindowsCancelUnblocksAWaitingRead(t *testing.T) {
	r, _, _ := winPipe(t)
	ch := winReadAsync(r, 8)
	time.Sleep(50 * time.Millisecond)
	r.Cancel()
	if got := winExpect(t, ch); !errors.Is(got.err, ErrCanceled) {
		t.Fatalf("Read error = %v, want ErrCanceled", got.err)
	}
}

func TestWindowsReadAfterCancelStaysCanceled(t *testing.T) {
	r, _, pw := winPipe(t)
	r.Cancel()
	r.Cancel() // idempotent
	if _, err := pw.Write([]byte("kept")); err != nil {
		t.Fatal(err)
	}
	if got := winExpect(t, winReadAsync(r, 8)); !errors.Is(got.err, ErrCanceled) {
		t.Fatalf("Read error = %v, want ErrCanceled", got.err)
	}
}

// A cancel must leave pending input in the pipe for the next reader.
func TestWindowsCancelDoesNotConsumeInput(t *testing.T) {
	r, pr, pw := winPipe(t)
	r.Cancel()
	if _, err := pw.Write([]byte("kept")); err != nil {
		t.Fatal(err)
	}
	winExpect(t, winReadAsync(r, 8))

	next, err := New(pr)
	if err != nil {
		t.Fatal(err)
	}
	if got := winExpect(t, winReadAsync(next, 8)); got.err != nil || string(got.b) != "kept" {
		t.Fatalf("next reader got %q, %v; want kept", got.b, got.err)
	}
}
