package spinner

import (
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

func TestStockFramesFollowTheThemesGlyphs(t *testing.T) {
	m := New()
	if got := ansi.StripANSI(m.View()); got != Frames()[0] {
		t.Errorf("default View = %q, want %q", got, Frames()[0])
	}
	m.Theme = theme.DarkTheme().ASCII()
	for i := 0; i < 8; i++ {
		got := ansi.StripANSI(m.View())
		if len(got) != 1 || got[0] >= 0x80 {
			t.Fatalf("frame %d under the ASCII theme = %q", i, got)
		}
		m.frame++
	}
}

func TestCustomFramesAreLeftAsGiven(t *testing.T) {
	m := New()
	m.Frames = []string{"●", "○"}
	m.Theme = theme.DarkTheme().ASCII()
	if got := ansi.StripANSI(m.View()); got != "●" {
		t.Errorf("custom frame = %q, want it unchanged", got)
	}
	m.Frames = nil
	m.Label = "loading"
	if got := m.View(); got != "loading" {
		t.Errorf("no frames = %q, want the label", got)
	}
}
