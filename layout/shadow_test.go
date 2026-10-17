package layout

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
)

func TestShadowAddsOneColumnAndRow(t *testing.T) {
	lines := strings.Split(NewBox().Border(NormalBorder()).ShadowGlyph("#", nil).render("ab"), "\n")
	want := []string{
		"┌──┐ ",
		"│ab│#",
		"└──┘#",
		" ####",
	}
	if strings.Join(lines, "\n") != strings.Join(want, "\n") {
		t.Fatalf("got\n%s\nwant\n%s", strings.Join(lines, "\n"), strings.Join(want, "\n"))
	}
}

func TestShadowWithMarginsAndNoBorder(t *testing.T) {
	out := strings.Split(NewBox().Margin(1, 2, 1, 2).ShadowGlyph("#", nil).render("ab"), "\n")
	for _, l := range out {
		if w := ansi.Width(l); w != 2+2+1+2 {
			t.Errorf("row %q is %d wide; want 7", l, w)
		}
	}
	if len(out) != 1+1+1+1 { // margin, box row, shadow row, margin
		t.Fatalf("%d rows: %q", len(out), out)
	}
	if !strings.HasPrefix(out[1], "  ab") {
		t.Fatalf("box row %q", out[1])
	}
	// The first row of a borderless box has a blank shadow cell: nothing to its right.
	solo := strings.Split(NewBox().ShadowGlyph("#", nil).render("ab"), "\n")
	if solo[0] != "ab " || solo[1] != " ##" {
		t.Fatalf("borderless: %q", solo)
	}
}

func TestShadowColourAndDefaultGlyph(t *testing.T) {
	out := NewBox().Border(NormalBorder()).Shadow(ansi.BrightBlack).render("x")
	if !strings.Contains(out, "▒") || !strings.Contains(out, ansi.NewStyle().Foreground(ansi.BrightBlack).Render("▒")) {
		t.Fatalf("no coloured shade: %q", out)
	}
	if NewBox().Border(NormalBorder()).Shadow(nil).ShadowGlyph("", nil).render("x") != NewBox().Border(NormalBorder()).render("x") {
		t.Fatal("an empty glyph did not remove the shadow")
	}
}
