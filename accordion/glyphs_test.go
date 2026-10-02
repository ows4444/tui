package accordion

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

func TestGlyphsFollowTheTheme(t *testing.T) {
	m := New(Section{Title: "One", Content: "body"}, Section{Title: "Two"})
	if v := ansi.StripANSI(m.View()); !strings.Contains(v, "▸ One") || !strings.Contains(v, "▸ Two") {
		t.Errorf("default View lost its markers: %q", v)
	}
	m.Theme = theme.DarkTheme().ASCII()
	v := ansi.StripANSI(m.View())
	for i := 0; i < len(v); i++ {
		if v[i] >= 0x80 {
			t.Fatalf("ASCII theme left non-ASCII output: %q", v)
		}
	}
	if !strings.Contains(v, "> One") || !strings.Contains(v, "> Two") {
		t.Errorf("ASCII markers missing: %q", v)
	}
}
