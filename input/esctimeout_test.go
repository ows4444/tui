package input

import (
	"io"
	"os"
	"testing"
	"time"
)

// readerKind builds a Reader over a writer end so tests run against both the
// read-deadline path (os.Pipe) and the goroutine fallback (io.Pipe).
func readerKinds(t *testing.T) map[string]func() (*Reader, io.Writer, func()) {
	return map[string]func() (*Reader, io.Writer, func()){
		"deadline": func() (*Reader, io.Writer, func()) {
			pr, pw, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			return NewReader(pr), pw, func() { pr.Close(); pw.Close() }
		},
		"fallback": func() (*Reader, io.Writer, func()) {
			pr, pw := io.Pipe()
			return NewReader(pr), pw, func() { pr.Close(); pw.Close() }
		},
	}
}

func readWithin(t *testing.T, rd *Reader) Event {
	t.Helper()
	type res struct {
		ev  Event
		err error
	}
	ch := make(chan res, 1)
	go func() { ev, err := rd.ReadEvent(); ch <- res{ev, err} }()
	select {
	case r := <-ch:
		if r.err != nil {
			t.Fatalf("err: %v", r.err)
		}
		return r.ev
	case <-time.After(3 * time.Second):
		t.Fatal("ReadEvent hung")
	}
	return nil
}

// #19
func TestEscSplitSequenceDecodesAsOneKey(t *testing.T) {
	for name, mk := range readerKinds(t) {
		t.Run(name, func(t *testing.T) {
			rd, w, done := mk()
			defer done()
			rd.SetEscTimeout(500 * time.Millisecond)
			go func() {
				w.Write([]byte{0x1b})
				time.Sleep(10 * time.Millisecond)
				w.Write([]byte("[A"))
				time.Sleep(10 * time.Millisecond)
				w.Write([]byte{0x1b})
				time.Sleep(10 * time.Millisecond)
				w.Write([]byte("OP"))
			}()
			if k, ok := readWithin(t, rd).(Key); !ok || k.Type != KeyUp {
				t.Fatalf("want Up, got %#v", k)
			}
			if k, ok := readWithin(t, rd).(Key); !ok || k.Type != KeyF1 {
				t.Fatalf("want F1, got %#v", k)
			}
		})
	}
}

// #20
func TestEscAloneEmitsEscAfterTimeout(t *testing.T) {
	for name, mk := range readerKinds(t) {
		t.Run(name, func(t *testing.T) {
			rd, w, done := mk()
			defer done()
			rd.SetEscTimeout(20 * time.Millisecond)
			go w.Write([]byte{0x1b})
			start := time.Now()
			if k, ok := readWithin(t, rd).(Key); !ok || k.Type != KeyEsc || k.Mod != 0 {
				t.Fatalf("want Esc, got %#v", k)
			}
			if time.Since(start) < 15*time.Millisecond {
				t.Fatalf("returned before timeout")
			}
			// A byte arriving after the timeout must not be lost or merged.
			go func() { time.Sleep(20 * time.Millisecond); w.Write([]byte("x")) }()
			if k, ok := readWithin(t, rd).(Key); !ok || k.Type != KeyRunes || k.Code != 'x' || k.Mod != 0 {
				t.Fatalf("want x, got %#v", k)
			}
		})
	}
}

// #21: a reader with no SetReadDeadline (io.Pipe) still honours the timeout.
func TestEscTimeoutWithoutReadDeadline(t *testing.T) {
	pr, pw := io.Pipe()
	defer pr.Close()
	defer pw.Close()
	if _, ok := any(pr).(interface{ SetReadDeadline(time.Time) error }); ok {
		t.Fatal("test premise: io.Pipe must lack SetReadDeadline")
	}
	rd := NewReader(pr)
	rd.SetEscTimeout(20 * time.Millisecond)
	go pw.Write([]byte{0x1b})
	if k, ok := readWithin(t, rd).(Key); !ok || k.Type != KeyEsc {
		t.Fatalf("want Esc, got %#v", k)
	}
	go func() { time.Sleep(5 * time.Millisecond); pw.Write([]byte("x")) }()
	if k, ok := readWithin(t, rd).(Key); !ok || k.Code != 'x' {
		t.Fatalf("byte lost after timeout: %#v", k)
	}
}

// #22
func TestEscTimeoutConfigurable(t *testing.T) {
	if DefaultEscTimeout < 25*time.Millisecond || DefaultEscTimeout > 50*time.Millisecond {
		t.Fatalf("default %v outside 25-50ms", DefaultEscTimeout)
	}
	pr, pw := io.Pipe()
	defer pr.Close()
	defer pw.Close()
	rd := NewReader(pr)
	if rd.EscTimeout() != DefaultEscTimeout {
		t.Fatalf("default not applied: %v", rd.EscTimeout())
	}
	rd.SetEscTimeout(0) // disabled: bare ESC is immediate
	go pw.Write([]byte{0x1b})
	start := time.Now()
	if k, ok := readWithin(t, rd).(Key); !ok || k.Type != KeyEsc {
		t.Fatalf("want Esc, got %#v", k)
	}
	if time.Since(start) > 15*time.Millisecond {
		t.Fatalf("timeout 0 should not wait")
	}
	rd.SetEscTimeout(80 * time.Millisecond)
	if rd.EscTimeout() != 80*time.Millisecond {
		t.Fatal("SetEscTimeout not stored")
	}
}
