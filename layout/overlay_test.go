package layout

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
)

func TestOverlayBasicPlacement(t *testing.T) {
	base := "aaaaa\naaaaa\naaaaa"
	overlay := "XX\nXX"
	got := Overlay(base, overlay, 1, 1)
	want := "aaaaa\naXXaa\naXXaa"
	if got != want {
		t.Errorf("Overlay() =\n%q\nwant\n%q", got, want)
	}
}

func TestOverlayAtOrigin(t *testing.T) {
	base := "aaaaa\naaaaa"
	overlay := "XX"
	got := Overlay(base, overlay, 0, 0)
	want := "XXaaa\naaaaa"
	if got != want {
		t.Errorf("Overlay() =\n%q\nwant\n%q", got, want)
	}
}

func TestOverlayOnlyTouchesTargetedRows(t *testing.T) {
	base := "row0\nrow1\nrow2\nrow3"
	overlay := "XX"
	got := Overlay(base, overlay, 0, 2)
	lines := strings.Split(got, "\n")
	if lines[0] != "row0" || lines[1] != "row1" || lines[3] != "row3" {
		t.Errorf("Overlay() untouched rows = %v, want row0/row1/row3 unchanged", lines)
	}
	if lines[2] != "XXw2" {
		t.Errorf("Overlay() targeted row = %q, want %q", lines[2], "XXw2")
	}
}

func TestOverlayRowsPastBaseAreDropped(t *testing.T) {
	base := "aaa\naaa"
	overlay := "X\nX\nX\nX" // 4 rows, only 1 fits starting at row 1
	got := Overlay(base, overlay, 0, 1)
	want := "aaa\nXaa"
	if got != want {
		t.Errorf("Overlay() =\n%q\nwant\n%q", got, want)
	}
}

func TestOverlayPadsShortBaseRow(t *testing.T) {
	base := "ab\nfull line"
	overlay := "XX"
	got := Overlay(base, overlay, 5, 0)
	want := "ab   XX\nfull line"
	if got != want {
		t.Errorf("Overlay() =\n%q\nwant\n%q", got, want)
	}
}

func TestOverlayPreservesContentPastRightEdge(t *testing.T) {
	base := "0123456789"
	overlay := "XX"
	got := Overlay(base, overlay, 2, 0)
	want := "01XX456789"
	if got != want {
		t.Errorf("Overlay() = %q, want %q", got, want)
	}
}

func TestOverlayWiderThanRemainingBaseExtendsLine(t *testing.T) {
	base := "ab"
	overlay := "XXXXXX"
	got := Overlay(base, overlay, 1, 0)
	want := "aXXXXXX"
	if got != want {
		t.Errorf("Overlay() = %q, want %q", got, want)
	}
}

func TestOverlayNegativeCoordsClampToZero(t *testing.T) {
	base := "aaaaa\naaaaa"
	overlay := "XX"
	got := Overlay(base, overlay, -3, -1)
	want := "XXaaa\naaaaa"
	if got != want {
		t.Errorf("Overlay() with negative x,y = %q, want %q (clamped to 0,0)", got, want)
	}
}

func TestOverlayWithStyledContent(t *testing.T) {
	base := ansi.NewStyle().Foreground(ansi.Blue).Render("background text here")
	overlay := ansi.NewStyle().Bold().Foreground(ansi.Red).Render("HI")

	got := Overlay(base, overlay, 5, 0)

	if !strings.Contains(got, overlay) {
		t.Errorf("Overlay() = %q, want it to contain the overlay's exact styled content %q", got, overlay)
	}
	if visible := ansi.StripANSI(got); visible != "backgHIund text here" {
		t.Errorf("StripANSI(Overlay()) = %q, want %q", visible, "backgHIund text here")
	}
}

func TestOverlayMultilineBox(t *testing.T) {
	base := strings.Repeat("background line that is long enough\n", 5)
	base = strings.TrimSuffix(base, "\n")
	overlay := "+--+\n|Hi|\n+--+"

	got := Overlay(base, overlay, 4, 1)
	lines := strings.Split(got, "\n")
	if len(lines) != 5 {
		t.Fatalf("got %d lines, want 5", len(lines))
	}
	if lines[0] != "background line that is long enough" {
		t.Errorf("row 0 (untouched) = %q", lines[0])
	}
	if !strings.HasPrefix(lines[1], "back+--+") {
		t.Errorf("row 1 = %q, want it to start with %q", lines[1], "back+--+")
	}
	if !strings.HasPrefix(lines[2], "back|Hi|") {
		t.Errorf("row 2 = %q, want it to start with %q", lines[2], "back|Hi|")
	}
	if !strings.HasPrefix(lines[3], "back+--+") {
		t.Errorf("row 3 = %q, want it to start with %q", lines[3], "back+--+")
	}
	if lines[4] != "background line that is long enough" {
		t.Errorf("row 4 (untouched) = %q", lines[4])
	}
}
