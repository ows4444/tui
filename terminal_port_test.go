package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui/internal/termio"
)

// Run reaches the terminal only through the port: with a fake that says the
// input is interactive, it enters raw mode, takes its size from the fake and
// leaves raw mode again, with no real terminal anywhere.
func TestRunUsesTheTerminalPort(t *testing.T) {
	pr, pw := mustPipe(t)
	defer pr.Close()
	defer pw.Close()
	ft := &termio.Fake{Interactive: true, W: 100, H: 30, SizeKnown: true}
	var out strings.Builder
	p := NewProgram(keyLogger{}, WithInput(pr), WithOutput(&out), WithMouse(MouseClick), WithTerminal(ft))
	done := make(chan error, 1)
	go func() { _, err := p.Run(); done <- err }()
	pw.Write([]byte("q"))
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return")
	}
	raw, unraw, _, _ := ft.Calls()
	if raw != 1 || unraw != 1 {
		t.Errorf("raw=%d unraw=%d, want 1 and 1", raw, unraw)
	}
	if !ft.LastMouse() {
		t.Error("MakeRaw was not told the mouse is on")
	}
	if p.width != 100 || p.height != 30 {
		t.Errorf("size = %dx%d, want 100x30 from the port", p.width, p.height)
	}
}

func TestRunRefusesANonInteractiveInput(t *testing.T) {
	pr, pw := mustPipe(t)
	defer pr.Close()
	defer pw.Close()
	p := NewProgram(keyLogger{}, WithInput(pr), WithOutput(&strings.Builder{}), WithTerminal(&termio.Fake{}))
	if _, err := p.Run(); err == nil || !strings.Contains(err.Error(), "not a terminal") {
		t.Fatalf("Run = %v, want an input-is-not-a-terminal error", err)
	}
}

func TestRunReturnsRawModeError(t *testing.T) {
	pr, pw := mustPipe(t)
	defer pr.Close()
	defer pw.Close()
	boom := errors.New("no raw")
	ft := &termio.Fake{Interactive: true, RawErr: boom}
	p := NewProgram(keyLogger{}, WithInput(pr), WithOutput(&strings.Builder{}), WithTerminal(ft))
	if _, err := p.Run(); !errors.Is(err, boom) {
		t.Fatalf("Run = %v, want %v", err, boom)
	}
}

// suspendErrModel suspends on 'e' and quits once the SuspendMsg arrives,
// keeping its Err.
type suspendErrModel struct {
	fn  func() error
	got *error
}

func (suspendErrModel) Init() Cmd { return nil }
func (m suspendErrModel) Update(msg Msg) (Model, Cmd) {
	switch msg := msg.(type) {
	case Key:
		if msg.Code == 'e' {
			return m, Suspend(m.fn)
		}
	case SuspendMsg:
		*m.got = msg.Err
		return m, Quit()
	}
	return m, nil
}
func (suspendErrModel) View() string { return "x" }

// runSuspend runs a Program on a fake terminal whose MakeRaw starts failing
// with rawErr during the Suspend, and returns the SuspendMsg's Err.
func runSuspend(t *testing.T, rawErr, fnErr error) error {
	t.Helper()
	pr, pw := mustPipe(t)
	defer pr.Close()
	defer pw.Close()
	ft := &termio.Fake{Interactive: true}
	var got error
	m := suspendErrModel{got: &got, fn: func() error {
		ft.RawErr = rawErr // read next by suspend, on this goroutine
		return fnErr
	}}
	p := NewProgram(m, WithInput(pr), WithOutput(&strings.Builder{}), WithTerminal(ft))
	errc := make(chan error, 1)
	go func() {
		_, err := p.Run()
		errc <- err
	}()
	if _, err := pw.Write([]byte("e")); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-errc:
		if err != nil {
			t.Fatalf("Run = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return")
	}
	return got
}

// When raw mode cannot be re-entered after a Suspend whose fn succeeded, the
// SuspendMsg says so; fn's own error is never replaced.
func TestSuspendReportsRawModeError(t *testing.T) {
	boom, fnErr := errors.New("no raw"), errors.New("editor failed")
	if err := runSuspend(t, boom, nil); !errors.Is(err, boom) {
		t.Errorf("fn ok, raw mode failed: SuspendMsg.Err = %v, want %v", err, boom)
	}
	if err := runSuspend(t, boom, fnErr); !errors.Is(err, fnErr) {
		t.Errorf("fn and raw mode failed: SuspendMsg.Err = %v, want %v", err, fnErr)
	}
	if err := runSuspend(t, nil, fnErr); !errors.Is(err, fnErr) {
		t.Errorf("fn failed: SuspendMsg.Err = %v, want %v", err, fnErr)
	}
	if err := runSuspend(t, nil, nil); err != nil {
		t.Errorf("nothing failed: SuspendMsg.Err = %v, want nil", err)
	}
}
