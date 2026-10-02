package widgets

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

func TestGradientCriteria(t *testing.T) {
	red := ansi.RGB{R: 255, G: 0, B: 0}
	green := ansi.RGB{R: 0, G: 255, B: 0}
	blue := ansi.RGB{R: 0, G: 0, B: 255}

	t.Run("empty text returns empty string (#495)", func(t *testing.T) {
		got := Gradient("", []ansi.RGB{red, blue}, false, theme.DarkTheme())
		if got != "" {
			t.Errorf("Gradient(\"\", ...) = %q, want \"\"", got)
		}
	})

	t.Run("single color renders uniformly (#494)", func(t *testing.T) {
		got := Gradient("abc", []ansi.RGB{red}, false, theme.DarkTheme())
		want := ansi.NewStyle().Foreground(red).Render("a") +
			ansi.NewStyle().Foreground(red).Render("b") +
			ansi.NewStyle().Foreground(red).Render("c")
		if got != want {
			t.Errorf("Gradient single color = %q, want %q", got, want)
		}
	})

	t.Run("interpolates per-character between two stops (#492)", func(t *testing.T) {
		text := "abc" // 3 chars: i=0 -> red, i=1 -> mid, i=2 -> blue
		got := Gradient(text, []ansi.RGB{red, blue}, false, theme.DarkTheme())

		wantFirst := ansi.NewStyle().Foreground(red).Render("a")
		wantLast := ansi.NewStyle().Foreground(blue).Render("c")
		if !strings.HasPrefix(got, wantFirst) {
			t.Errorf("first char not styled with first stop color: got %q", got)
		}
		if !strings.HasSuffix(got, wantLast) {
			t.Errorf("last char not styled with last stop color: got %q", got)
		}

		// Middle char (i=1, n=3) should be exactly halfway: (127 or 128, 0, 127 or 128).
		mid := gradientColorAt(1, 3, []ansi.RGB{red, blue})
		if mid.R < 120 || mid.R > 135 || mid.B < 120 || mid.B > 135 || mid.G != 0 {
			t.Errorf("mid color not plausibly interpolated: %+v", mid)
		}
		wantMid := ansi.NewStyle().Foreground(mid).Render("b")
		if !strings.Contains(got, wantMid) {
			t.Errorf("Gradient output missing expected mid-color segment; got %q want substring %q", got, wantMid)
		}

		// Each character should be styled individually, not per-line: the
		// three per-character renders concatenated should equal the whole
		// output.
		wantWhole := wantFirst + wantMid + wantLast
		if got != wantWhole {
			t.Errorf("Gradient(%q) = %q, want %q", text, got, wantWhole)
		}
	})

	t.Run("stretches interpolation across text longer than stops (#493)", func(t *testing.T) {
		colors := []ansi.RGB{red, green, blue}
		text := "abcde" // 5 chars, 3 stops -> stretched, not clamped/repeated

		got := Gradient(text, colors, false, theme.DarkTheme())

		wantFirst := ansi.NewStyle().Foreground(red).Render("a")
		wantLast := ansi.NewStyle().Foreground(blue).Render("e")
		if !strings.HasPrefix(got, wantFirst) {
			t.Errorf("first char not styled with first stop color: got %q", got)
		}
		if !strings.HasSuffix(got, wantLast) {
			t.Errorf("last char not styled with last stop color: got %q", got)
		}

		// Middle char (i=2 of 5) should land exactly on the middle stop
		// (green), proving proportional stretching across all stops
		// rather than only ever interpolating between the first two.
		midColor := gradientColorAt(2, 5, colors)
		if midColor != green {
			t.Errorf("gradientColorAt(2, 5, ...) = %+v, want %+v (green stop)", midColor, green)
		}

		// And it must differ from a naive "repeat/clamp to nearest stop"
		// implementation for an interior, non-stop-aligned index.
		c1 := gradientColorAt(1, 5, colors)
		if c1 == red || c1 == green {
			t.Errorf("gradientColorAt(1, 5, ...) = %+v, expected a proportional blend, not a clamped stop", c1)
		}
	})

	t.Run("bold applies on top of interpolated colors (#496)", func(t *testing.T) {
		got := Gradient("ab", []ansi.RGB{red, blue}, true, theme.DarkTheme())
		want := ansi.NewStyle().Bold().Foreground(red).Render("a") +
			ansi.NewStyle().Bold().Foreground(blue).Render("b")
		if got != want {
			t.Errorf("Gradient bold = %q, want %q", got, want)
		}
	})

	t.Run("no colors leaves text unstyled without panicking", func(t *testing.T) {
		got := Gradient("abc", nil, false, theme.DarkTheme())
		if got != "abc" {
			t.Errorf("Gradient with no colors = %q, want %q", got, "abc")
		}
	})
}

func TestLerpByte(t *testing.T) {
	tests := []struct {
		a, b uint8
		t    float64
		want uint8
	}{
		{0, 255, 0, 0},
		{0, 255, 1, 255},
		{0, 100, 0.5, 50},
		{100, 0, 0.5, 50},
		{10, 10, 0.7, 10},
	}
	for _, tt := range tests {
		got := lerpByte(tt.a, tt.b, tt.t)
		if got != tt.want {
			t.Errorf("lerpByte(%d, %d, %v) = %d, want %d", tt.a, tt.b, tt.t, got, tt.want)
		}
	}
}
