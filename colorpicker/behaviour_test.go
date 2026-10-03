package colorpicker

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
)

func TestSetThemeReplacesTheme(t *testing.T) {
	m := New(ansi.Red).SetTheme(theme.LightTheme())
	if m.Theme != theme.LightTheme() {
		t.Fatalf("Theme = %+v, want theme.Light", m.Theme)
	}
}

func TestParseHexRejectsBadInput(t *testing.T) {
	for _, s := range []string{"", "#12345", "123456x", "#gg0000", "#00gg00", "#0000gg", "#12345678"} {
		if _, ok := parseHex(s); ok {
			t.Errorf("parseHex(%q) ok, want rejected", s)
		}
	}
	got, ok := parseHex("#0aFf10")
	if !ok || got != (ansi.RGB{R: 0x0a, G: 0xff, B: 0x10}) {
		t.Errorf("parseHex(#0aFf10) = %v, %v", got, ok)
	}
}

func TestEnterOnEmptyPaletteAndNonKeyMsgAreInert(t *testing.T) {
	m := New()
	if _, cmd := m.Update(tui.Key{Type: tui.KeyEnter}); cmd != nil {
		t.Error("Enter on an empty palette returned a Cmd")
	}
	if _, cmd := m.Update(struct{}{}); cmd != nil {
		t.Error("non-key Msg returned a Cmd")
	}
}

func TestInvalidHexEnterIsNoOp(t *testing.T) {
	m := New(ansi.Red)
	m, _ = m.Update(tui.Key{Type: tui.KeyTab})
	m.HexInput.SetValue("#zz")
	if _, cmd := m.Update(tui.Key{Type: tui.KeyEnter}); cmd != nil {
		t.Error("invalid hex confirmed a colour")
	}
}

func TestMeasureMatchesRenderedView(t *testing.T) {
	m := New(ansi.Red, ansi.Green)
	sz := m.LayoutNode().Measure(layout.Loose(layout.Size{W: 80, H: 5}))
	if want := ansi.Width(m.View()); sz.W != want || sz.H != 1 {
		t.Fatalf("Measure = %+v, want %dx1", sz, want)
	}
}

func TestRenderZeroSizeIsEmpty(t *testing.T) {
	if out := New(ansi.Red).LayoutNode().Render(layout.Size{W: 0, H: 3}); out != "" {
		t.Fatalf("Render at zero width = %q, want empty", out)
	}
	if out := New(ansi.Red).LayoutNode().Render(layout.Size{W: 10, H: 0}); out != "" {
		t.Fatalf("Render at zero height = %q, want empty", out)
	}
}

func TestEmptyPaletteLayoutGivesHexInputTheWidth(t *testing.T) {
	n := New().LayoutNode().(pickerNode)
	if n.swatchesWidth() != 0 {
		t.Fatalf("swatchesWidth = %d, want 0", n.swatchesWidth())
	}
	exact(t, n.Render(layout.Size{W: 20, H: 1}), 20, 1)
}

func TestDescribeNamesEveryColourKind(t *testing.T) {
	for c, want := range map[ansi.Color]string{
		ansi.BasicColor(1):         "red",
		ansi.BasicColor(99):        "custom colour",
		ansi.Color256(200):         "colour 200",
		ansi.RGB{R: 1, G: 2, B: 3}: "#010203",
	} {
		if got := describe(c); got != want {
			t.Errorf("describe(%v) = %q, want %q", c, got, want)
		}
	}
	if got := describe(nil); got != "custom colour" {
		t.Errorf("describe(nil) = %q", got)
	}
}

func TestLinearizeEmptyPaletteSaysSo(t *testing.T) {
	if out := New().Linearize(); !strings.HasPrefix(out, "No colours\nHex:") {
		t.Fatalf("Linearize = %q", out)
	}
}

// A paste reaches the hex field while it has focus, and is ignored otherwise.
func TestPasteGoesToTheFocusedHexField(t *testing.T) {
	m := New(nil)
	next, _ := m.Update(tui.PasteEvent{Text: "#ff0000"})
	if next.HexInput.Value() != "" {
		t.Fatalf("a paste with the palette focused changed the hex field to %q", next.HexInput.Value())
	}
	m, _ = m.Update(tui.Key{Type: tui.KeyTab})
	m, _ = m.Update(tui.PasteEvent{Text: "#ff0000"})
	if m.HexInput.Value() != "#ff0000" {
		t.Fatalf("hex field after a paste = %q, want #ff0000", m.HexInput.Value())
	}
}
