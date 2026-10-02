package theme

import (
	"math"
	"testing"

	"github.com/ows4444/tui/ansi"
)

// basicColorRGB gives each of the 16 named ANSI colors a canonical RGB
// value (xterm's own documented default palette) purely so this test can
// compute a contrast ratio for Dark/Light, which predate the RGB-based
// presets and use ansi.BasicColor. What a real terminal actually displays
// for e.g. ansi.Red depends on that terminal's own color scheme — this is
// a reference point for catching a badly-chosen pairing, not a guarantee
// of the rendered result.
var basicColorRGB = map[ansi.BasicColor][3]uint8{
	ansi.Black:         {0x00, 0x00, 0x00},
	ansi.Red:           {0xCD, 0x00, 0x00},
	ansi.Green:         {0x00, 0xCD, 0x00},
	ansi.Yellow:        {0xCD, 0xCD, 0x00},
	ansi.Blue:          {0x00, 0x00, 0xEE},
	ansi.Magenta:       {0xCD, 0x00, 0xCD},
	ansi.Cyan:          {0x00, 0xCD, 0xCD},
	ansi.White:         {0xE5, 0xE5, 0xE5},
	ansi.BrightBlack:   {0x7F, 0x7F, 0x7F},
	ansi.BrightRed:     {0xFF, 0x00, 0x00},
	ansi.BrightGreen:   {0x00, 0xFF, 0x00},
	ansi.BrightYellow:  {0xFF, 0xFF, 0x00},
	ansi.BrightBlue:    {0x5C, 0x5C, 0xFF},
	ansi.BrightMagenta: {0xFF, 0x00, 0xFF},
	ansi.BrightCyan:    {0x00, 0xFF, 0xFF},
	ansi.BrightWhite:   {0xFF, 0xFF, 0xFF},
}

func rgbOf(t *testing.T, c ansi.Color) (r, g, b uint8) {
	t.Helper()
	switch v := c.(type) {
	case ansi.RGB:
		return v.R, v.G, v.B
	case ansi.BasicColor:
		rgb, ok := basicColorRGB[v]
		if !ok {
			t.Fatalf("no canonical RGB entry for BasicColor %d", v)
		}
		return rgb[0], rgb[1], rgb[2]
	default:
		t.Fatalf("rgbOf: unsupported color type %T (add a case if a preset starts using it)", c)
		return 0, 0, 0
	}
}

// relativeLuminance implements the WCAG 2.x relative luminance formula
// (https://www.w3.org/TR/WCAG21/#dfn-relative-luminance).
func relativeLuminance(r, g, b uint8) float64 {
	lin := func(c uint8) float64 {
		cs := float64(c) / 255
		if cs <= 0.03928 {
			return cs / 12.92
		}
		return math.Pow((cs+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(r) + 0.7152*lin(g) + 0.0722*lin(b)
}

// contrastRatio implements the WCAG 2.x contrast ratio formula
// (https://www.w3.org/TR/WCAG21/#dfn-contrast-ratio): (L1+0.05)/(L2+0.05)
// with L1 the lighter of the two relative luminances.
func contrastRatio(t *testing.T, fg, bg ansi.Color) float64 {
	t.Helper()
	fr, fgc, fb := rgbOf(t, fg)
	br, bgc, bb := rgbOf(t, bg)
	l1 := relativeLuminance(fr, fgc, fb)
	l2 := relativeLuminance(br, bgc, bb)
	if l1 < l2 {
		l1, l2 = l2, l1
	}
	return (l1 + 0.05) / (l2 + 0.05)
}

// minTextContrastRatio is WCAG AA's threshold for normal-size text.
const minTextContrastRatio = 4.5

// TestPresetTextContrast asserts every built-in preset's primary text
// color reads clearly against its own background: Text is each preset's
// normal-text foreground, and TextInverse doubles as that preset's base
// background color (see the "// background" comments in presets.go, and
// widgets/badge.go's Background(accent)+Foreground(TextInverse)
// convention, which only makes sense if TextInverse is "the color that
// reads against a filled background," i.e. the theme's own background
// tone). A preset added later that fails this is illegible by
// construction, not just aesthetically off.
func TestPresetTextContrast(t *testing.T) {
	presets := map[string]Theme{
		"Dark":           DarkTheme(),
		"Light":          LightTheme(),
		"Dracula":        DraculaTheme(),
		"Nord":           NordTheme(),
		"Gruvbox":        GruvboxTheme(),
		"TokyoNight":     TokyoNightTheme(),
		"Monokai":        MonokaiTheme(),
		"SolarizedDark":  SolarizedDarkTheme(),
		"SolarizedLight": SolarizedLightTheme(),
		"Catppuccin":     CatppuccinTheme(),
		"OneDark":        OneDarkTheme(),
		"NightOwl":       NightOwlTheme(),
		"RosePine":       RosePineTheme(),
		"EverforestDark": EverforestDarkTheme(),
		"GitHubDark":     GitHubDarkTheme(),
		"HighContrast":   HighContrastTheme(),
	}
	for name, th := range presets {
		t.Run(name, func(t *testing.T) {
			ratio := contrastRatio(t, th.Text, th.TextInverse)
			if ratio < minTextContrastRatio {
				t.Errorf("%s: Text/TextInverse contrast ratio = %.2f, want >= %.1f (WCAG AA normal text)",
					name, ratio, minTextContrastRatio)
			}
		})
	}
}

// minAAAContrastRatio is WCAG AAA's threshold for normal-size text.
const minAAAContrastRatio = 7.0

// TestHighContrastMeetsAAA holds every colour token of HighContrast to WCAG
// AAA against its own background, so the preset cannot quietly regress into
// an ordinary theme.
func TestHighContrastMeetsAAA(t *testing.T) {
	tokens := map[string]ansi.Color{
		"Text": HighContrastTheme().Text, "Primary": HighContrastTheme().Primary,
		"Secondary": HighContrastTheme().Secondary, "Success": HighContrastTheme().Success,
		"Warning": HighContrastTheme().Warning, "Error": HighContrastTheme().Error,
		"Info": HighContrastTheme().Info, "Muted": HighContrastTheme().Muted,
		"BorderColor": HighContrastTheme().BorderColor, "Focus": HighContrastTheme().Focus,
		"Selection": HighContrastTheme().Selection,
	}
	for name, c := range tokens {
		if ratio := contrastRatio(t, c, HighContrastTheme().TextInverse); ratio < minAAAContrastRatio {
			t.Errorf("HighContrast.%s contrast = %.2f, want >= %.1f (WCAG AAA)", name, ratio, minAAAContrastRatio)
		}
	}
}

// TestHighContrastStatusColoursAreDistinct checks meaning is not carried by
// near-identical colours: Success, Warning and Error must all differ, in
// both hue and lightness terms (compared by RGB distance).
func TestHighContrastStatusColoursAreDistinct(t *testing.T) {
	status := map[string]ansi.Color{"Success": HighContrastTheme().Success, "Warning": HighContrastTheme().Warning, "Error": HighContrastTheme().Error}
	names := []string{"Success", "Warning", "Error"}
	for i := 0; i < len(names); i++ {
		for j := i + 1; j < len(names); j++ {
			r1, g1, b1 := rgbOf(t, status[names[i]])
			r2, g2, b2 := rgbOf(t, status[names[j]])
			d := math.Hypot(math.Hypot(float64(r1)-float64(r2), float64(g1)-float64(g2)), float64(b1)-float64(b2))
			if d < 100 {
				t.Errorf("%s and %s are too close (RGB distance %.0f)", names[i], names[j], d)
			}
		}
	}
	// It must also actually be the highest-contrast preset: pure white on black.
	if got := contrastRatio(t, HighContrastTheme().Text, HighContrastTheme().TextInverse); got < 20.9 {
		t.Errorf("HighContrast Text/background = %.2f, want the maximum 21:1", got)
	}
}
