package render

import (
	"testing"

	"github.com/ows4444/tui/ansi"
)

const (
	alef = "א"
	bet  = "ב"
	gml  = "ג"
)

var meas = ansi.ClusterMeasurer(true)

func TestBidiLineReordersRightToLeftText(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{alef + bet + gml, gml + bet + alef},                                     // all RTL: base level 1, whole line reversed
		{"abc " + alef + bet + gml + " def", "abc " + gml + bet + alef + " def"}, // an RTL run in LTR text
		{alef + bet + " abc", "abc " + bet + alef},                               // RTL paragraph with an LTR run
		{"x 12 " + alef + bet, "x 12 " + bet + alef},                             // numbers stay left to right
	} {
		if got := BidiLine(c.in, meas); got != c.want {
			t.Errorf("BidiLine(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// A line with nothing to reorder comes back unchanged, byte for byte: plain
// text, styled text, links, graphics and everything the cell parser declines.
func TestBidiLineLeavesOtherLinesAlone(t *testing.T) {
	for _, in := range []string{
		"", "plain ascii", "café 中文 ──",
		"\x1b[31mred\x1b[0m text", "\x1b]8;;https://e.com\x1b\\link\x1b]8;;\x1b\\",
		"\x1b_Ga=T;AAAA\x1b\\ image", "\x1bPqsixel\x1b\\",
		"\x1b[31m" + "é" + "\x1b[0m",
	} {
		if got := BidiLine(in, meas); got != in {
			t.Errorf("BidiLine(%q) = %q, want it unchanged", in, got)
		}
	}
}

func TestBidiLineKeepsEachCharactersStyleAndLink(t *testing.T) {
	in := "ab \x1b[31m" + alef + bet + "\x1b[0m \x1b]8;;https://e.com\x1b\\" + gml + "\x1b]8;;\x1b\\ cd"
	got := BidiLine(in, meas)
	plain := ansi.StripANSI(got)
	if want := "ab " + gml + " " + bet + alef + " cd"; plain != want {
		t.Fatalf("text = %q, want %q", plain, want)
	}
	// alef and bet stay red wherever they land, and gimel keeps its link.
	glyphs, _, ok := ParseGlyphs(got, meas)
	if !ok {
		t.Fatal("the output does not parse")
	}
	for _, g := range glyphs {
		switch g.Text {
		case alef, bet:
			if g.Style.FG == 0 {
				t.Errorf("%q lost its colour", g.Text)
			}
		case gml:
			if g.Style.Link != "https://e.com" {
				t.Errorf("gimel lost its link: %+v", g.Style)
			}
		case "a", "b", "c", "d":
			if g.Style != (Style{}) {
				t.Errorf("%q gained a style: %+v", g.Text, g.Style)
			}
		}
	}
	if last := glyphs[len(glyphs)-1]; last.Style != (Style{}) {
		t.Errorf("a style leaked to the end of the line: %+v", last.Style)
	}
}

func TestBidiLineMirrorsBracketsInRightToLeftRuns(t *testing.T) {
	got := BidiLine(alef+"("+bet+")", meas)
	// Logical alef ( bet ): the whole line is RTL, so it reads right to left and
	// the parentheses are mirrored: visually ( bet ) alef... shown as ")" bet "(" alef reversed.
	want := "(" + bet + ")" + alef
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestBidiLineKeepsWideCharactersWhole(t *testing.T) {
	got := BidiLine(alef+"中"+bet, meas)
	if got != bet+"中"+alef {
		t.Errorf("got %q", got)
	}
	g, _, ok := ParseGlyphs(got, meas)
	if !ok || len(g) != 4 { // bet, wide head, its continuation, alef
		t.Errorf("glyphs = %d (ok %v), want 4", len(g), ok)
	}
}
