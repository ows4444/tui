package tui

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
)

const bidiView = "abc \u05d0\u05d1\u05d2 def\n\u05d0\u05d1 (x)\nplain line"

func frameOf(t *testing.T, view string, opts ...ProgramOption) string {
	t.Helper()
	out, read := captureOutput(t)
	opts = append([]ProgramOption{WithOutput(out), WithAltScreen(false)}, opts...)
	p := NewProgram(staticModel{view: view}, opts...)
	p.render()
	return string(read())
}

// Criterion #66: with WithBidi each line is drawn in UAX #9 visual order.
func TestWithBidiDrawsLinesInVisualOrder(t *testing.T) {
	for name, opt := range map[string]ProgramOption{"cell": WithCellRenderer(true), "line": WithCellRenderer(false)} {
		got := ansi.StripANSI(frameOf(t, bidiView, WithBidi(true), opt))
		for _, want := range []string{"abc \u05d2\u05d1\u05d0 def", "(x) \u05d1\u05d0", "plain line"} {
			if !strings.Contains(got, want) {
				t.Errorf("%s renderer: frame lacks %q in %q", name, want, got)
			}
		}
	}
}

// Criterion #67: with the option off, or on for a line with nothing to
// reorder, the bytes are exactly those of a Program without it.
func TestWithoutBidiOutputIsByteIdentical(t *testing.T) {
	for _, view := range []string{bidiView, "plain\nlines \u00e9\u4e2d", "\x1b[31mred\x1b[0m"} {
		base := frameOf(t, view)
		if off := frameOf(t, view, WithBidi(false)); off != base {
			t.Errorf("WithBidi(false) changed the frame for %q:\n got %q\nwant %q", view, off, base)
		}
	}
	for _, view := range []string{"plain\nlines \u00e9\u4e2d", "\x1b[31mred\x1b[0m"} {
		if on := frameOf(t, view, WithBidi(true)); on != frameOf(t, view) {
			t.Errorf("WithBidi(true) changed a line with nothing to reorder: %q", view)
		}
	}
	if frameOf(t, bidiView, WithBidi(true)) == frameOf(t, bidiView) {
		t.Error("WithBidi(true) changed nothing for a view with Hebrew")
	}
}
