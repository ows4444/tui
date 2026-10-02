package braille

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

func TestDotCellBrailleAndASCII(t *testing.T) {
	if got := DotCell(Bit[0][0]|Bit[1][3], theme.UnicodeGlyphSet()); got != string(rune(0x2800+0x01+0x80)) {
		t.Fatalf("braille cell = %q", got)
	}
	ascii := theme.ASCIIGlyphSet()
	if got := DotCell(0, ascii); got != " " {
		t.Fatalf("empty ASCII cell = %q, want a space", got)
	}
	if got := DotCell(0xFF, ascii); got == " " || got == "" {
		t.Fatalf("full ASCII cell = %q, want the densest shade", got)
	}
}

func TestArcKeepsOnlyTheAnnulus(t *testing.T) {
	fill, track := ansi.NewStyle(), ansi.NewStyle()
	cells := Arc(8, 2, 8, 8, 8, 4, func(px, py, d float64) bool { return px < 0 }, fill, track, theme.UnicodeGlyphSet())
	if len(cells) != 2 || len(cells[0]) != 8 {
		t.Fatalf("grid is %dx%d, want 2 rows of 8", len(cells), len(cells[0]))
	}
	flat := ""
	for _, r := range cells {
		flat += strings.Join(r, "")
	}
	if !strings.ContainsRune(flat, ' ') {
		t.Error("no blank cells: the inside of the ring was drawn")
	}
	if !strings.ContainsFunc(flat, func(r rune) bool { return r >= 0x2800 && r <= 0x28FF }) {
		t.Error("no braille cells drawn")
	}
}

func TestArcMinWidth(t *testing.T) {
	if ArcMinWidth != 8 {
		t.Fatalf("ArcMinWidth = %d; the Gauge and ProgressCircle label geometry assumes 8", ArcMinWidth)
	}
}
