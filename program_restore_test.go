package tui

import (
	"fmt"
	"io"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/ows4444/tui/ansi"
)

// spinModel re-renders continuously: every Update returns a Cmd that yields
// the next spinMsg immediately.
type spinModel struct{ n int }

type spinMsg struct{}

func (m spinModel) Init() Cmd { return func() Msg { return spinMsg{} } }
func (m spinModel) Update(msg Msg) (Model, Cmd) {
	if _, ok := msg.(spinMsg); ok {
		m.n++
		return m, func() Msg { return spinMsg{} }
	}
	return m, nil
}
func (m spinModel) View() string { return fmt.Sprintf("frame %d", m.n) }

// unsyncBuffer is deliberately not goroutine-safe: under -race, any two
// unserialised writers to it are reported.
type unsyncBuffer struct{ b []byte }

func (u *unsyncBuffer) Write(p []byte) (int, error) { u.b = append(u.b, p...); return len(p), nil }

func startSpinning(t *testing.T, opts ...ProgramOption) (*Program, *unsyncBuffer, <-chan error) {
	t.Helper()
	pr, pw := io.Pipe()
	t.Cleanup(func() { _ = pw.Close() })
	out := &unsyncBuffer{}
	opts = append([]ProgramOption{
		WithInput(pr), WithOutput(out), WithMaxFPS(0),
	}, opts...)
	p := NewProgram(spinModel{}, opts...)
	errc := make(chan error, 1)
	go func() { _, err := p.Run(); errc <- err }()
	// Wait until frames are flowing.
	deadline := time.Now().Add(5 * time.Second)
	for {
		p.outMu.Lock()
		n := len(out.b)
		p.outMu.Unlock()
		if n > 200 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("no frames rendered")
		}
		time.Sleep(time.Millisecond)
	}
	return p, out, errc
}

func waitRun(t *testing.T, errc <-chan error) error {
	t.Helper()
	select {
	case err := <-errc:
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return")
		return nil
	}
}

// When SIGHUP arrives while frames are rendered continuously, there is no data
// race (run with -race) and no frame bytes follow the restore sequence.
func TestSIGHUPDuringContinuousRenderingIsRaceFreeAndRestoresLast(t *testing.T) {
	_, out, errc := startSpinning(t, WithExitOnSignal(false))
	if err := syscall.Kill(syscall.Getpid(), syscall.SIGHUP); err != nil {
		t.Fatal(err)
	}
	if err := waitRun(t, errc); err != ErrInterrupted {
		t.Fatalf("Run = %v, want ErrInterrupted", err)
	}
	got := string(out.b)
	if !strings.HasSuffix(got, ansi.AltScreenDisable) {
		t.Fatalf("output does not end with the restore sequence; tail %q", got[max(0, len(got)-80):])
	}
	if strings.Count(got, ansi.AltScreenDisable) != 1 {
		t.Fatalf("restore sequence written %d times", strings.Count(got, ansi.AltScreenDisable))
	}
}

// When a Cmd panics while a frame is being written, the panic handler's
// restore (restoreViaLoop) leaves no frame bytes after the restore sequence.
func TestCmdPanicRestoreDuringRenderingLeavesNoTrailingFrame(t *testing.T) {
	p, out, errc := startSpinning(t)
	p.restoreViaLoop() // what dispatch/runSequence/runEvery do on recover()
	p.Send(QuitMsg{})
	if err := waitRun(t, errc); err != nil {
		t.Fatalf("Run = %v", err)
	}
	got := string(out.b)
	if !strings.HasSuffix(got, ansi.AltScreenDisable) {
		t.Fatalf("output does not end with the restore sequence; tail %q", got[max(0, len(got)-80):])
	}
}

// With no loop draining, the handoff times out and the fixed restore string is
// written once under the output mutex; later writes are dropped.
func TestRestoreViaLoopFallsBackToFixedString(t *testing.T) {
	out := &unsyncBuffer{}
	p := NewProgram(spinModel{}, WithOutput(out), WithBracketedPaste(true))
	start := time.Now()
	p.restoreViaLoop()
	if time.Since(start) > 3*time.Second {
		t.Fatal("fallback took too long")
	}
	want := p.restoreFixedString()
	if want == "" || string(out.b) != want {
		t.Fatalf("output %q, want fixed restore %q", out.b, want)
	}
	p.write("late frame")
	p.restoreViaLoop()
	if string(out.b) != want {
		t.Fatalf("bytes written after restore: %q", out.b)
	}
}
