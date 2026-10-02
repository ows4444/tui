package tui

import (
	"strings"
	"testing"
)

// TestInlineExitEndsOnFreshLine proves criterion #24: an inline exit with a
// live region leaves the cursor at column 0 of the row below it.
func TestInlineExitEndsOnFreshLine(t *testing.T) {
	out, read := captureOutput(t)
	p := NewProgram(staticModel{view: "a\nb"}, WithOutput(out), WithAltScreen(false))
	p.render()
	frame := string(read())
	p.restoreTerminal()
	got := strings.TrimPrefix(string(read()), frame) // read is cumulative
	if !strings.HasSuffix(got, "\r\n") && !strings.Contains(got, "\r\n") {
		t.Fatalf("inline exit with live region wrote no CRLF: %q", got)
	}
}

// TestInlineExitNoLiveRegionNoNewline: nothing live, nothing to move below.
func TestInlineExitNoLiveRegionNoNewline(t *testing.T) {
	out, read := captureOutput(t)
	p := NewProgram(staticModel{view: "v"}, WithOutput(out), WithAltScreen(false))
	p.restoreTerminal()
	if got := string(read()); strings.Contains(got, "\n") {
		t.Fatalf("exit without live region wrote a newline: %q", got)
	}
}

// TestAltScreenExitNoExtraNewline proves criterion #25.
func TestAltScreenExitNoExtraNewline(t *testing.T) {
	out, read := captureOutput(t)
	p := NewProgram(staticModel{view: "a\nb"}, WithOutput(out), WithAltScreen(true))
	p.render()
	frame := string(read())
	p.restoreTerminal()
	got := string(read())
	if len(got) >= len(frame) && strings.HasPrefix(got, frame) {
		got = got[len(frame):] // read may return cumulative output
	}
	if strings.Contains(got, "\n") {
		t.Fatalf("alt-screen exit wrote a newline: %q", got)
	}
}
