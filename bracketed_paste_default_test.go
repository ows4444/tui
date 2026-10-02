package tui

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
)

// #16: with no option, mode 2004 is on and every restore path turns it off.
func TestBracketedPasteIsOnByDefault(t *testing.T) {
	p := NewProgram(staticModel{view: "v"})
	if !p.bracketedPaste {
		t.Fatal("bracketedPaste is off by default")
	}
	if !strings.Contains(p.restoreFixedString(), ansi.BracketedPasteDisable) {
		t.Errorf("fixed restore (signal and timeout path) %q lacks %q", p.restoreFixedString(), ansi.BracketedPasteDisable)
	}

	out, read := captureOutput(t)
	p = NewProgram(staticModel{view: "v"}, WithOutput(out))
	p.enterModes()
	p.leaveModes() // the quit and suspend path
	got := string(read())
	if strings.Count(got, ansi.BracketedPasteEnable) != 1 || strings.Count(got, ansi.BracketedPasteDisable) != 1 {
		t.Fatalf("enter/leave wrote %q, want one enable and one disable", got)
	}
	if strings.Index(got, ansi.BracketedPasteEnable) > strings.Index(got, ansi.BracketedPasteDisable) {
		t.Errorf("disable before enable: %q", got)
	}
}

// #17: WithBracketedPaste(false) writes neither code.
func TestBracketedPasteOptOutWritesNothing(t *testing.T) {
	out, read := captureOutput(t)
	p := NewProgram(staticModel{view: "v"}, WithOutput(out), WithBracketedPaste(false))
	p.enterModes()
	p.leaveModes()
	if got := string(read()); strings.Contains(got, "\x1b[?2004") {
		t.Fatalf("output %q mentions mode 2004", got)
	}
	if s := p.restoreFixedString(); strings.Contains(s, "\x1b[?2004") {
		t.Fatalf("fixed restore %q mentions mode 2004", s)
	}
}
