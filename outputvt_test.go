package tui

import (
	"errors"
	"strings"
	"testing"

	"github.com/ows4444/tui/internal/termio"
)

func TestRunEnablesAndRestoresOutputVT(t *testing.T) {
	ft := &termio.Fake{}
	p := NewProgram(keyLogger{},
		WithInput(strings.NewReader("q")),
		WithOutput(mustDevNull(t)),
		WithAltScreen(false),
		WithTerminal(ft),
	)
	if _, err := runReaderProgram(t, p); err != nil {
		t.Fatal(err)
	}
	if _, _, enabled, restored := ft.Calls(); enabled != 1 || restored != 1 {
		t.Fatalf("enabled=%d restored=%d, want 1 and 1", enabled, restored)
	}
}

// restoreTerminal is what the panic defer and the signal handler both run.
func TestRestoreTerminalRestoresOutputVTOnce(t *testing.T) {
	ft := &termio.Fake{}
	p := NewProgram(keyLogger{}, WithOutput(mustDevNull(t)), WithAltScreen(false), WithTerminal(ft))
	if err := p.enterOutputVT(); err != nil {
		t.Fatal(err)
	}
	func() {
		defer func() { _ = recover() }()
		defer p.restoreTerminal() // the deferred path of a panicking Run
		panic("boom")
	}()
	p.restoreTerminal() // the signal path racing it
	if _, _, _, restored := ft.Calls(); restored != 1 {
		t.Fatalf("restored=%d, want exactly 1", restored)
	}
}

func TestRunReturnsErrorWhenOutputVTFails(t *testing.T) {
	boom := errors.New("no vt")
	p := NewProgram(keyLogger{},
		WithInput(strings.NewReader("q")),
		WithOutput(mustDevNull(t)),
		WithTerminal(&termio.Fake{VTErr: boom}),
	)
	_, err := p.Run()
	if !errors.Is(err, boom) || !strings.Contains(err.Error(), "virtual terminal processing") {
		t.Fatalf("Run = %v, want error naming virtual terminal processing", err)
	}
}
