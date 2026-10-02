package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/internal/capprobe"
)

// A DECRPM answer for mode 2026 that says "not recognised" (0) or "permanently
// reset" (4) must end with no synchronized-output brackets in any frame; one
// that says "set" (1) or "reset but settable" (2) keeps them.
func TestMode2026ReplyDecidesWhetherFramesAreBracketed(t *testing.T) {
	for _, c := range []struct {
		reply      string
		wantBrackt bool
	}{
		{"?2026;0$y", false},
		{"?2026;4$y", false},
		{"?2026;1$y", true},
		{"?2026;2$y", true},
	} {
		var parser capprobe.Parser
		parser.Observe('[', c.reply)
		_, done, res := parser.Observe('[', "?62;22c")
		if !done {
			t.Fatalf("%s: DA1 did not end the probe", c.reply)
		}
		out, read := captureOutput(t)
		p := NewProgram(staticModel{view: "x"}, WithOutput(out), WithAltScreen(false), WithCapabilityProbe(time.Hour))
		p.capProbe.resolve(Capabilities{SyncOutput: res.SyncOutput})
		p.render()
		got := strings.Contains(string(read()), "?2026")
		if got != c.wantBrackt {
			t.Errorf("reply %s: frame has 2026 brackets = %v, want %v", c.reply, got, c.wantBrackt)
		}
	}
}

// NO_COLOR wins over COLORTERM and CLICOLOR_FORCE in the default (cell)
// renderer, not only in the line renderer.
func TestNoColorBeatsColortermInTheCellRenderer(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	t.Setenv("COLORTERM", "truecolor")
	t.Setenv("CLICOLOR_FORCE", "1")
	out, read := captureOutput(t)
	p := NewProgram(staticModel{view: truecolorView}, WithOutput(out), WithAltScreen(false))
	if p.ColorProfile() != ansi.NoColor {
		t.Fatalf("ColorProfile = %v, want NoColor", p.ColorProfile())
	}
	p.render()
	got := string(read())
	for _, code := range []string{"38;2", "48;2", "38;5", "48;5", ";31", "[31", "[34"} {
		if strings.Contains(got, code) {
			t.Errorf("colour code %q leaked: %q", code, got)
		}
	}
	if !strings.Contains(got, "red") || !strings.Contains(got, "blue") {
		t.Errorf("text lost: %q", got)
	}
}
