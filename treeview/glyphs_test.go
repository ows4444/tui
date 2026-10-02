package treeview

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

func asciiOnly(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}

// Under the ASCII theme the markers are ASCII; by default they are unchanged.
func TestGlyphsFollowTheTheme(t *testing.T) {
	m := testTree()
	if v := ansi.StripANSI(m.View()); !strings.Contains(v, "▸ root") {
		t.Errorf("default View lost its marker: %q", v)
	}
	m.Theme = theme.DarkTheme().ASCII()
	collapsed := ansi.StripANSI(m.View())
	next, _ := m.Update(tui.Key{Type: tui.KeyEnter}) // expand root
	expanded := ansi.StripANSI(next.View())
	for _, v := range []string{collapsed, expanded} {
		if !asciiOnly(v) {
			t.Errorf("ASCII theme left non-ASCII output: %q", v)
		}
	}
	if !strings.Contains(collapsed, "> root") && !strings.Contains(collapsed, "> > root") {
		t.Errorf("collapsed marker missing: %q", collapsed)
	}
	if !strings.Contains(expanded, "v root") {
		t.Errorf("expanded marker missing: %q", expanded)
	}
}
