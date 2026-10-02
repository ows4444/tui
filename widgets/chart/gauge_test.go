package chart

import (
	"math"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

func isBraille(r rune) bool { return r >= 0x2800 && r <= 0x28FF }

// arcCells counts braille cells rendered in colour c. Gauge styles each
// arc cell as its own span, so counting opening sequences that are
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

func TestGaugeZeroOrNegativeWidth(t *testing.T) {
	for _, w := range []int{0, -3} {
		if got := Gauge(0.5, w, theme.DarkTheme()); got != "" {
			t.Errorf("Gauge(0.5, %d) = %q, want empty", w, got)
		}
	}
}

func TestGaugeClamps(t *testing.T) {
	for _, w := range []int{5, 20} {
		zero := Gauge(0, w, theme.DarkTheme())
		one := Gauge(1, w, theme.DarkTheme())
		for name, p := range map[string]float64{"negative": -0.5, "-inf": math.Inf(-1), "nan": math.NaN()} {
			if got := Gauge(p, w, theme.DarkTheme()); got != zero {
				t.Errorf("width %d: Gauge(%s) differs from Gauge(0)", w, name)
			}
		}
		for name, p := range map[string]float64{"above one": 1.7, "+inf": math.Inf(1)} {
			if got := Gauge(p, w, theme.DarkTheme()); got != one {
				t.Errorf("width %d: Gauge(%s) differs from Gauge(1)", w, name)
			}
		}
	}
}

func TestGaugeGeometry(t *testing.T) {
	for _, w := range []int{8, 9, 10, 20, 33, 60} {
		base := strings.Split(Gauge(0, w, theme.DarkTheme()), "\n")
		for _, p := range []float64{0, 0.01, 0.25, 0.5, 0.999, 1} {
			lines := strings.Split(Gauge(p, w, theme.DarkTheme()), "\n")
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

func TestGaugeShowsPercentLabel(t *testing.T) {
	tests := []struct {
		percent float64
		want    string
	}{
		{0, "0%"}, {0.42, "42%"}, {0.425, "43%"}, {1, "100%"}, {2, "100%"},
	}
	for _, w := range []int{8, 20, 40} {
		for _, tt := range tests {
			plain := ansi.StripANSI(Gauge(tt.percent, w, theme.DarkTheme()))
			if !strings.Contains(plain, tt.want) {
				t.Errorf("width %d Gauge(%v) missing label %q:\n%s", w, tt.percent, tt.want, plain)
			}
		}
	}
}

func TestGaugeFillAndTrackColours(t *testing.T) {
	for _, w := range []int{8, 20, 40} {
		th := theme.DarkTheme()
		if n := arcCells(Gauge(0, w, th), th.Primary); n != 0 {
			t.Errorf("width %d 0%%: %d Primary cells, want 0", w, n)
		}
		if n := arcCells(Gauge(0, w, th), th.Muted); n == 0 {
			t.Errorf("width %d 0%%: no Muted track cells", w)
		}
		if n := arcCells(Gauge(1, w, th), th.Muted); n != 0 {
			t.Errorf("width %d 100%%: %d Muted cells, want 0", w, n)
		}
		if n := arcCells(Gauge(1, w, th), th.Primary); n == 0 {
			t.Errorf("width %d 100%%: no Primary cells", w)
		}

		prev := -1
		for i := 0; i <= 100; i++ {
			n := arcCells(Gauge(float64(i)/100, w, th), th.Primary)
			if n < prev {
				t.Errorf("width %d: Primary cells dropped %d -> %d at %d%%", w, prev, n, i)
			}
			prev = n
		}
	}
}

func TestGaugeArcIsSymmetricRing(t *testing.T) {
	// At 100% the arc is a half ring: the top row must be non-empty and
	// the label row's centre must be free of arc cells.
	lines := strings.Split(ansi.StripANSI(Gauge(1, 20, theme.DarkTheme())), "\n")
	if strings.TrimSpace(lines[0]) == "" {
		t.Errorf("top row empty:\n%s", strings.Join(lines, "\n"))
	}
	last := []rune(lines[len(lines)-1])
	mid := last[len(last)/2-2 : len(last)/2+2]
	for _, r := range mid {
		if isBraille(r) {
			t.Errorf("arc cell inside label area: %q", string(last))
		}
	}
}

func TestGaugeNarrowFallback(t *testing.T) {
	for w := 1; w < 8; w++ {
		got := Gauge(0.5, w, theme.DarkTheme())
		if strings.Contains(got, "\n") {
			t.Errorf("width %d: fallback has multiple lines", w)
		}
		if gw := ansi.Width(got); gw != w {
			t.Errorf("width %d: fallback Width = %d", w, gw)
		}
	}
}

func TestGaugeUsesOnlyThemeColours(t *testing.T) {
	th := theme.DarkTheme()
	th.Primary, th.Muted, th.Text = ansi.Red, ansi.Green, ansi.Yellow
	out := Gauge(0.5, 20, th)
	for _, c := range []ansi.Color{ansi.Red, ansi.Green, ansi.Yellow} {
		seq := strings.TrimSuffix(ansi.NewStyle().Foreground(c).Render("x"), "x"+ansi.Reset)
		if !strings.Contains(out, seq) {
			t.Errorf("expected colour %q in output", seq)
		}
	}
}
