package colorpicker

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
)

var _ tui.Linearizer = Model{}

func TestLinearize(t *testing.T) {
	m := New(ansi.Red, ansi.BrightBlue, ansi.Color256(202), ansi.RGB{R: 0x12, G: 0xab, B: 0xef})
	got := m.Linearize()
	lines := strings.Split(got, "\n")
	want := []string{
		"red, swatch 1 of 4, selected",
		"bright blue, swatch 2 of 4",
		"colour 202, swatch 3 of 4",
		"#12abef, swatch 4 of 4",
	}
	for i, w := range want {
		if lines[i] != w {
			t.Errorf("line %d = %q, want %q", i, lines[i], w)
		}
	}
	if !strings.HasPrefix(lines[4], "Hex: ") {
		t.Errorf("hex line = %q", lines[4])
	}
	m.hexFocused = true
	if strings.Contains(m.Linearize(), "selected") {
		t.Error("no swatch is selected while the hex field has focus")
	}
	if got := New().Linearize(); !strings.HasPrefix(got, "No colours") {
		t.Errorf("empty = %q", got)
	}
}
