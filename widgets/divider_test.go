package widgets

import (
	"testing"

	"github.com/ows4444/tui/ansi"
)

func TestDivider(t *testing.T) {
	tests := []struct {
		width int
		want  string
	}{
		{5, "─────"},
		{1, "─"},
		{0, ""},
		{-1, ""},
	}
	for _, tt := range tests {
		if got := Divider(tt.width); got != tt.want {
			t.Errorf("Divider(%d) = %q, want %q", tt.width, got, tt.want)
		}
		if got := ansi.Width(Divider(tt.width)); tt.width > 0 && got != tt.width {
			t.Errorf("Width(Divider(%d)) = %d, want %d", tt.width, got, tt.width)
		}
	}
}

func TestDividerLabelCentersWithEvenRemainder(t *testing.T) {
	// " Section " is 9 wide; width 20 leaves 11 to split 5/6.
	got := DividerLabel(20, "Section")
	want := "─────" + " Section " + "──────"
	if got != want {
		t.Errorf("DividerLabel(20, %q) = %q, want %q", "Section", got, want)
	}
	if w := ansi.Width(got); w != 20 {
		t.Errorf("Width(DividerLabel(20, ...)) = %d, want 20", w)
	}
}

func TestDividerLabelCentersWithOddRemainder(t *testing.T) {
	// " Section " is 9 wide; width 21 leaves 12 to split 6/6.
	got := DividerLabel(21, "Section")
	want := "──────" + " Section " + "──────"
	if got != want {
		t.Errorf("DividerLabel(21, %q) = %q, want %q", "Section", got, want)
	}
}

func TestDividerLabelTruncatesWhenTooNarrow(t *testing.T) {
	got := DividerLabel(5, "Section")
	want := " Sect" // ansi.Truncate(" Section ", 5)
	if got != want {
		t.Errorf("DividerLabel(5, %q) = %q, want %q", "Section", got, want)
	}
}

func TestDividerLabelZeroWidth(t *testing.T) {
	if got := DividerLabel(0, "Section"); got != "" {
		t.Errorf("DividerLabel(0, ...) = %q, want empty", got)
	}
}
