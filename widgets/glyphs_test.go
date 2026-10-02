package widgets

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

func TestProgressBarGlyphsFollowTheTheme(t *testing.T) {
	// Default output is unchanged.
	if got := ansi.StripANSI(ProgressBar(0.5, 10, theme.DarkTheme())); got != strings.Repeat("█", 5)+strings.Repeat("░", 5) {
		t.Errorf("default bar = %q", got)
	}
	// An explicitly Unicode theme is byte-identical to the zero one.
	u := theme.DarkTheme()
	u.Glyphs = theme.UnicodeGlyphSet()
	if ProgressBar(0.3, 12, u) != ProgressBar(0.3, 12, theme.DarkTheme()) {
		t.Error("UnicodeGlyphs differs from the default")
	}
	// ASCII theme: only ASCII, same shape.
	if got := ansi.StripANSI(ProgressBar(0.5, 10, theme.DarkTheme().ASCII())); got != "#####-----" {
		t.Errorf("ASCII bar = %q", got)
	}
	// Custom glyphs and full / empty extremes.
	c := theme.DarkTheme()
	c.Glyphs = theme.Glyphs{BarFull: "=", BarEmpty: "."}
	if got := ansi.StripANSI(ProgressBar(1, 4, c)); got != "====" {
		t.Errorf("full custom bar = %q", got)
	}
	if got := ansi.StripANSI(ProgressBar(0, 4, c)); got != "...." {
		t.Errorf("empty custom bar = %q", got)
	}
}
