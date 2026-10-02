package maskedinput

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

func TestLayoutNodeNeverDrawsTheValue(t *testing.T) {
	m := New()
	m.Prompt = "Card: "
	m.SetValue("4111111111111111")
	out := ansi.StripANSI(m.LayoutNode().Render(layout.Size{W: 40, H: 1}))
	if strings.Contains(out, "4111") || !strings.Contains(out, "•") {
		t.Errorf("rendered %q", out)
	}
	if !strings.Contains(ansi.StripANSI(m.Model.LayoutNode().Render(layout.Size{W: 40, H: 1})), "4111111111111111") {
		t.Fatal("test premise changed: the embedded LayoutNode no longer draws the value")
	}
}
