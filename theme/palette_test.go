package theme

import (
	"testing"

	"github.com/ows4444/tui/ansi"
)

type fakePalette struct{ i, r, g, b uint8 }

func (f fakePalette) PaletteColor() (i, r, g, b uint8) { return f.i, f.r, f.g, f.b }

func TestPaletteReportedRGBUsedForContrast(t *testing.T) {
	p := NewPalette()
	fg, bg := ansi.BasicColor(7), ansi.BasicColor(0)
	base := Contrast(fg, bg)
	if got := p.Contrast(fg, bg); got != base {
		t.Fatalf("unanswered palette = %v, want xterm %v", got, base)
	}
	// Terminal reports colour 7 as a dark grey close to colour 0.
	if !p.Observe(fakePalette{7, 0x10, 0x10, 0x10}) || !p.Reported(7) || p.Reported(0) {
		t.Fatal("Observe/Reported wrong")
	}
	if got := p.Contrast(fg, bg); got >= base || got != Contrast(ansi.RGB{R: 0x10, G: 0x10, B: 0x10}, ansi.RGB{}) {
		t.Errorf("reported contrast = %v (xterm %v)", got, base)
	}
	if got := p.Contrast(ansi.Color256(7), bg); got >= base {
		t.Errorf("Color256(7) ignored the palette: %v", got)
	}
	if p.Observe("nope") {
		t.Error("Observe accepted a foreign message")
	}
}

func TestPaletteCheckUsesReportedColours(t *testing.T) {
	th := ForBackground(ansi.RGB{})
	th.TextInverse = ansi.RGB{}
	th.Text = ansi.BasicColor(7)
	p := NewPalette()
	if len(p.Check(th, 4.5)) != len(th.Check(4.5)) {
		t.Error("unanswered palette should match Theme.Check")
	}
	p.Set(7, 0x10, 0x10, 0x10)
	found := false
	for _, is := range p.Check(th, 4.5) {
		found = found || is.Role == "Text"
	}
	if !found {
		t.Error("Text should fail against the reported palette")
	}
	var nilP *Palette
	if nilP.Contrast(ansi.BasicColor(7), ansi.BasicColor(0)) != Contrast(ansi.BasicColor(7), ansi.BasicColor(0)) {
		t.Error("nil palette should be xterm")
	}
	p.Set(99, 1, 2, 3) // ignored
}
