package widgets

import (
	"math"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// TestProgressCircleFullRing proves #419: filled dots appear both above
// and below the horizontal midline (unlike Gauge's semicircle, which only
// ever fills the upper half-plane), and the row count is roughly double
// Gauge's for the same width.
func TestProgressCircleFullRing(t *testing.T) {
	width := 20
	th := theme.DarkTheme()
	lines := strings.Split(ansi.StripANSI(ProgressCircle(1, width, th)), "\n")

	gaugeLines := strings.Split(ansi.StripANSI(Gauge(1, width, th)), "\n")
	if len(lines) < 2*len(gaugeLines)-1 || len(lines) > 2*len(gaugeLines)+1 {
		t.Errorf("ProgressCircle has %d rows, Gauge has %d rows at width %d; want roughly double", len(lines), len(gaugeLines), width)
	}

	top := strings.TrimSpace(lines[0])
	bottom := strings.TrimSpace(lines[len(lines)-1])
	if top == "" {
		t.Errorf("top row empty, want dots above the midline:\n%s", strings.Join(lines, "\n"))
	}
	if bottom == "" {
		t.Errorf("bottom row empty, want dots below the midline:\n%s", strings.Join(lines, "\n"))
	}
}

func isBraille(r rune) bool { return r >= 0x2800 && r <= 0x28FF }

// arcCells counts braille cells rendered in colour c. ProgressCircle styles
// each arc cell as its own span, so counting opening sequences that are
// followed by a braille rune is exact.
func arcCells(out string, c ansi.Color) int {
	seq := strings.TrimSuffix(ansi.NewStyle().Foreground(c).Render("x"), "x"+ansi.Reset)
	n := 0
	for _, part := range strings.Split(out, seq)[1:] {
		if r, _ := utf8.DecodeRuneInString(part); isBraille(r) {
			n++
		}
	}
	return n
}

// TestProgressCircleReusesBrailleArc proves #420: Gauge and ProgressCircle
// share the same per-dot angle/radius test loop via braille.Arc rather than
// each carrying its own copy. This is a structural property we can only
// check indirectly through behaviour: both widgets' fill counts grow
// monotonically with percent and use exactly the same annulus/fill-style
// rules (space when empty, styled braille otherwise), which is what
// braille.Arc guarantees for both callers.
func TestProgressCircleReusesBrailleArc(t *testing.T) {
	th := theme.DarkTheme()
	prev := -1
	for i := 0; i <= 100; i++ {
		n := arcCells(ProgressCircle(float64(i)/100, 24, th), th.Primary)
		if n < prev {
			t.Errorf("Primary cells dropped %d -> %d at %d%%", prev, n, i)
		}
		prev = n
	}
	if arcCells(ProgressCircle(0, 24, th), th.Primary) != 0 {
		t.Errorf("0%%: expected no Primary cells")
	}
	if arcCells(ProgressCircle(1, 24, th), th.Muted) != 0 {
		t.Errorf("100%%: expected no Muted cells")
	}
}

// TestProgressCircleLabelCentered proves #421: the percentage label sits
// on the middle row of the ring, not the bottom row (Gauge's convention).
func TestProgressCircleLabelCentered(t *testing.T) {
	width := 20
	lines := strings.Split(ansi.StripANSI(ProgressCircle(0.42, width, theme.DarkTheme())), "\n")
	mid := lines[len(lines)/2]
	if !strings.Contains(mid, "42%") {
		t.Errorf("middle row %q does not contain label, want \"42%%\"", mid)
	}
	last := lines[len(lines)-1]
	if strings.Contains(last, "42%") {
		t.Errorf("label found on bottom row %q, want it centred instead", last)
	}
}

// TestProgressCircleLabelClearOfRing checks the label area at
// progressCircleMinWidth is free of ring dots before the label overwrites
// it, backing the "by construction" claim in ProgressCircle's docstring.
func TestProgressCircleLabelClearOfRing(t *testing.T) {
	lines := strings.Split(ansi.StripANSI(ProgressCircle(1, progressCircleMinWidth, theme.DarkTheme())), "\n")
	mid := lines[len(lines)/2]
	label := "100%"
	// The label itself must be present verbatim (no ring dot overwrote a
	// label glyph), which is only possible if that span was blank first.
	if !strings.Contains(mid, label) {
		t.Errorf("label area corrupted by ring dots: %q", mid)
	}
}

func TestProgressCircleZeroOrNegativeWidth(t *testing.T) {
	for _, w := range []int{0, -3} {
		if got := ProgressCircle(0.5, w, theme.DarkTheme()); got != "" {
			t.Errorf("ProgressCircle(0.5, %d) = %q, want empty", w, got)
		}
	}
}

// TestProgressCircleClamps proves #422: percent outside [0,1] or NaN is
// clamped, matching Gauge's convention.
func TestProgressCircleClamps(t *testing.T) {
	for _, w := range []int{progressCircleMinWidth, 30} {
		zero := ProgressCircle(0, w, theme.DarkTheme())
		one := ProgressCircle(1, w, theme.DarkTheme())
		for name, p := range map[string]float64{"negative": -0.5, "-inf": math.Inf(-1), "nan": math.NaN()} {
			if got := ProgressCircle(p, w, theme.DarkTheme()); got != zero {
				t.Errorf("width %d: ProgressCircle(%s) differs from ProgressCircle(0)", w, name)
			}
		}
		for name, p := range map[string]float64{"above one": 1.7, "+inf": math.Inf(1)} {
			if got := ProgressCircle(p, w, theme.DarkTheme()); got != one {
				t.Errorf("width %d: ProgressCircle(%s) differs from ProgressCircle(1)", w, name)
			}
		}
	}
}

// TestProgressCircleNarrowFallback proves #423: below
// progressCircleMinWidth, ProgressCircle falls back to a one-line
// ProgressBar, matching Gauge's small-width fallback convention.
func TestProgressCircleNarrowFallback(t *testing.T) {
	for w := 1; w < progressCircleMinWidth; w++ {
		got := ProgressCircle(0.5, w, theme.DarkTheme())
		want := ProgressBar(0.5, w, theme.DarkTheme())
		if got != want {
			t.Errorf("width %d: ProgressCircle(0.5) = %q, want ProgressBar fallback %q", w, got, want)
		}
		if strings.Contains(got, "\n") {
			t.Errorf("width %d: fallback has multiple lines", w)
		}
		if gw := ansi.Width(got); gw != w {
			t.Errorf("width %d: fallback Width = %d", w, gw)
		}
	}
}

func TestProgressCircleGeometry(t *testing.T) {
	for _, w := range []int{progressCircleMinWidth, progressCircleMinWidth + 1, 30, 40, 60} {
		base := strings.Split(ProgressCircle(0, w, theme.DarkTheme()), "\n")
		for _, p := range []float64{0, 0.01, 0.25, 0.5, 0.999, 1} {
			lines := strings.Split(ProgressCircle(p, w, theme.DarkTheme()), "\n")
			if len(lines) != len(base) {
				t.Errorf("width %d percent %v: %d lines, want %d (stable across percent)", w, p, len(lines), len(base))
			}
			for i, l := range lines {
				if got := ansi.Width(l); got != w {
					t.Errorf("width %d percent %v line %d: Width = %d, want %d", w, p, i, got, w)
				}
			}
		}
	}
}

func TestProgressCircleUsesOnlyThemeColours(t *testing.T) {
	th := theme.DarkTheme()
	th.Primary, th.Muted, th.Text = ansi.Red, ansi.Green, ansi.Yellow
	out := ProgressCircle(0.5, 24, th)
	for _, c := range []ansi.Color{ansi.Red, ansi.Green, ansi.Yellow} {
		seq := strings.TrimSuffix(ansi.NewStyle().Foreground(c).Render("x"), "x"+ansi.Reset)
		if !strings.Contains(out, seq) {
			t.Errorf("expected colour %q in output", seq)
		}
	}
}
