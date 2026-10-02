package tui

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
)

// A non-empty NO_COLOR disables colour by default, per https://no-color.org.
func TestNoColorEnvironmentVariableDisablesColourByDefault(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	out, read := captureOutput(t)
	p := NewProgram(staticModel{view: truecolorView}, WithOutput(out), WithCellRenderer(false))
	if p.ColorProfile() != ansi.NoColor {
		t.Fatalf("ColorProfile = %v with NO_COLOR=1, want NoColor", p.ColorProfile())
	}
	p.render()
	got := string(read())
	for _, code := range []string{"38;2", "48;2", "38;5", "48;5"} {
		if strings.Contains(got, code) {
			t.Errorf("colour code %q leaked with NO_COLOR set: %q", code, got)
		}
	}
	// Attributes other than colour are kept, as with WithColorProfile(NoColor).
	if !strings.Contains(got, "\x1b[1mred\x1b[0m blue\x1b[0m") {
		t.Errorf("unexpected NO_COLOR frame: %q", got)
	}
}

// An explicit WithColorProfile wins over NO_COLOR, even ansi.TrueColor, and
// whatever the option order.
func TestExplicitColorProfileOverridesNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	for _, prof := range []ansi.Profile{ansi.TrueColor, ansi.ANSI256, ansi.ANSI16, ansi.NoColor} {
		p := NewProgram(staticModel{view: "x"}, WithColorProfile(prof))
		if p.ColorProfile() != prof {
			t.Errorf("WithColorProfile(%v) under NO_COLOR gave %v", prof, p.ColorProfile())
		}
		p = NewProgram(staticModel{view: "x"}, WithAltScreen(false), WithColorProfile(prof), WithAltScreen(false))
		if p.ColorProfile() != prof {
			t.Errorf("option order changed the result for %v: %v", prof, p.ColorProfile())
		}
	}
}

// Only a non-empty value counts: an empty NO_COLOR leaves output unchanged.
func TestEmptyNoColorLeavesTheDefaultUnchanged(t *testing.T) {
	t.Setenv("CLICOLOR_FORCE", "1") // output is a regular file, not a TTY
	t.Setenv("NO_COLOR", "")
	t.Setenv("COLORTERM", "truecolor")
	out, read := captureOutput(t)
	p := NewProgram(staticModel{view: truecolorView}, WithOutput(out), WithCellRenderer(false))
	if p.ColorProfile() != ansi.TrueColor {
		t.Fatalf("ColorProfile = %v with NO_COLOR empty, want TrueColor", p.ColorProfile())
	}
	p.render()
	if !strings.Contains(string(read()), truecolorView) {
		t.Error("empty NO_COLOR altered the output")
	}
}
