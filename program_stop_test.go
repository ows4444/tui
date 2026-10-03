package tui

import (
	"context"
	"errors"
	"io"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ows4444/tui/ansi"
)

// lockedOut is a writer a test can read while the Program writes to it.
type lockedOut struct {
	mu sync.Mutex
	b  strings.Builder
}

func (l *lockedOut) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.Write(p)
}

func (l *lockedOut) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.String()
}

// runUntil starts p and returns its error once Run has returned.
func runUntil(t *testing.T, p *Program, trigger func()) error {
	t.Helper()
	errc := make(chan error, 1)
	go func() {
		_, err := p.Run()
		errc <- err
	}()
	for end := time.Now().Add(3 * time.Second); !p.running(); {
		if time.Now().After(end) {
			t.Fatal("Run never started its loop")
		}
		time.Sleep(time.Millisecond)
	}
	trigger()
	select {
	case err := <-errc:
		return err
	case <-time.After(3 * time.Second):
		t.Fatal("Run did not return")
		return nil
	}
}

// Cancelling the context given to WithContext ends Run: the terminal is
// restored and Run returns the context's error.
func TestCancellingTheParentContextEndsRun(t *testing.T) {
	pr, pw := io.Pipe()
	defer pw.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	out := &lockedOut{}
	p := NewProgram(staticModel{view: "x"}, WithInput(pr), WithOutput(out), WithContext(ctx), WithAltScreen(true))

	err := runUntil(t, p, cancel)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run = %v, want context.Canceled", err)
	}
	got := out.String()
	if !strings.Contains(got, ansi.AltScreenEnable) || !strings.HasSuffix(got, ansi.AltScreenDisable) {
		t.Fatalf("the terminal was not restored last; tail %q", got[max(0, len(got)-60):])
	}
	if p.Context().Err() == nil {
		t.Error("Program.Context is not done after Run returned")
	}
}

// The same for a deadline on the parent.
func TestParentContextDeadlineEndsRun(t *testing.T) {
	pr, pw := io.Pipe()
	defer pw.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	p := NewProgram(staticModel{view: "x"}, WithInput(pr), WithOutput(io.Discard), WithContext(ctx))
	if err := runUntil(t, p, func() {}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Run = %v, want context.DeadlineExceeded", err)
	}
}

// failingOut writes normally until fail is set, then returns err.
type failingOut struct {
	fail atomic.Bool
	err  error
}

func (f *failingOut) Write(p []byte) (int, error) {
	if f.fail.Load() {
		return 0, f.err
	}
	return len(p), nil
}

// countView shows a number that each int Msg replaces, so every one repaints.
type countView struct{ n int }

func (countView) Init() Cmd { return nil }
func (m countView) Update(msg Msg) (Model, Cmd) {
	if n, ok := msg.(int); ok {
		m.n = n
	}
	return m, nil
}
func (m countView) View() string { return "n=" + strconv.Itoa(m.n) }

// When a write to the output fails, Run returns that error instead of going
// on rendering to a writer that is gone.
func TestOutputWriteErrorEndsRun(t *testing.T) {
	pr, pw := io.Pipe()
	defer pw.Close()
	boom := errors.New("connection closed")
	out := &failingOut{err: boom}
	p := NewProgram(countView{}, WithInput(pr), WithOutput(out))

	err := runUntil(t, p, func() {
		out.fail.Store(true)
		p.Send(1) // the repaint is the write that fails
	})
	if !errors.Is(err, boom) {
		t.Fatalf("Run = %v, want an error wrapping %v", err, boom)
	}
}
