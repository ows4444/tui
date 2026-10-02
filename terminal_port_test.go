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
