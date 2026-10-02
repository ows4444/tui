package numberinput

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

func TestLayoutNodeExactSize(t *testing.T) {
	m := New()
	m.Prompt = "> "
	m.Placeholder = "hint"
	filled := m
	filled.SetValue("12345678901234567890123456789012345678901234567890")
	for name, mm := range map[string]Model{"empty": m, "long": filled} {
		if got := mm.LayoutNode().Measure(layout.Unconstrained()); got.H != 1 || got.W < 3 {
			t.Errorf("%s Measure = %v", name, got)
		}
		for _, s := range []layout.Size{{W: 1, H: 1}, {W: 20, H: 3}, {W: 80, H: 24}} {
			lines := strings.Split(mm.LayoutNode().Render(s), "\n")
			if len(lines) != s.H {
				t.Fatalf("%s %v: %d rows", name, s, len(lines))
			}
			for _, l := range lines {
				if ansi.Width(l) != s.W {
					t.Errorf("%s %v: row width %d (%q)", name, s, ansi.Width(l), l)
				}
			}
		}
	}
}
