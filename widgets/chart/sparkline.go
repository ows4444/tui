package chart

import "github.com/ows4444/tui/theme"

// sparkBars is the default (Unicode) height scale, lowest to highest.
var sparkBars = []rune(theme.UnicodeGlyphSet().Sparks)

// Sparkline renders values as a single line of block characters, scaled
// between the slice's own min and max. A flat slice (min == max, including
// a single value) renders at a uniform middle height rather than full
// height, since "no variance" and "at the max" are different things worth
// looking different. An empty slice renders as "".
func Sparkline(values []float64) string { return SparklineWith(values, theme.Theme{}) }

// SparklineWith is Sparkline drawn with t's glyphs (theme.Glyphs.Sparks), so an
// ASCII theme gets ASCII bars. Sparkline is SparklineWith on the default theme.
func SparklineWith(values []float64, t theme.Theme) string {
	if len(values) == 0 {
		return ""
	}
	bars := []rune(t.GlyphSet().Sparks)

	min, max := values[0], values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	span := max - min

	runes := make([]rune, len(values))
	for i, v := range values {
		if span == 0 {
			runes[i] = bars[len(bars)/2]
			continue
		}
		frac := (v - min) / span
		level := int(frac*float64(len(bars)-1) + 0.5) // round to nearest
		if level < 0 {
			level = 0
		}
		if level >= len(bars) {
			level = len(bars) - 1
		}
		runes[i] = bars[level]
	}
	return string(runes)
}
