package theme

import (
	"testing"

	"github.com/ows4444/tui/ansi"
)

// paletteRGB resolves a colour of any depth ForProfile can produce to RGB,
// using the xterm palette for Color256 (and the same reference values as
// contrast_test.go for the 16 basic colours).
func paletteRGB(t *testing.T, c ansi.Color) ansi.RGB {
	t.Helper()
	if v, ok := c.(ansi.Color256); ok {
		n := int(v)
		switch {
		case n < 16:
			c = ansi.BasicColor(n)
		case n < 232:
			n -= 16
			lv := [6]uint8{0, 95, 135, 175, 215, 255}
			return ansi.RGB{R: lv[n/36], G: lv[(n/6)%6], B: lv[n%6]}
		default:
			g := uint8(8 + 10*(n-232))
			return ansi.RGB{R: g, G: g, B: g}
		}
	}
	r, g, b := rgbOf(t, c)
	return ansi.RGB{R: r, G: g, B: b}
}

// TestContrastAfterForProfile records, for every preset under ANSI256 and
// ANSI16, the lowest contrast ratio of any colour token against the preset's
// background (TextInverse), and fails if HighContrast drops below AA.
func TestContrastAfterForProfile(t *testing.T) {
	presets := []struct {
		name string
		th   Theme
	}{
		{"Dark", DarkTheme()}, {"Light", LightTheme()}, {"Dracula", DraculaTheme()}, {"Nord", NordTheme()},
		{"Gruvbox", GruvboxTheme()}, {"TokyoNight", TokyoNightTheme()}, {"Monokai", MonokaiTheme()},
		{"SolarizedDark", SolarizedDarkTheme()}, {"SolarizedLight", SolarizedLightTheme()},
		{"Catppuccin", CatppuccinTheme()}, {"OneDark", OneDarkTheme()}, {"NightOwl", NightOwlTheme()},
		{"RosePine", RosePineTheme()}, {"EverforestDark", EverforestDarkTheme()},
		{"GitHubDark", GitHubDarkTheme()}, {"HighContrast", HighContrastTheme()},
	}
	profiles := []struct {
		name string
		p    ansi.Profile
	}{{"ANSI256", ansi.ANSI256}, {"ANSI16", ansi.ANSI16}}

	for _, pr := range presets {
		for _, pf := range profiles {
			d := pr.th.ForProfile(pf.p)
			tokens := []struct {
				name string
				c    ansi.Color
			}{
				{"Primary", d.Primary}, {"Secondary", d.Secondary}, {"Success", d.Success},
				{"Warning", d.Warning}, {"Error", d.Error}, {"Info", d.Info},
				{"Muted", d.Muted}, {"Text", d.Text}, {"BorderColor", d.BorderColor},
				{"Focus", d.Focus}, {"Selection", d.Selection},
			}
			bg := paletteRGB(t, d.TextInverse)
			lowest, lowestName := 1e9, ""
			for _, tk := range tokens {
				if r := contrastRatio(t, paletteRGB(t, tk.c), bg); r < lowest {
					lowest, lowestName = r, tk.name
				}
			}
			t.Logf("%-14s %-8s lowest contrast %.2f:1 (%s)", pr.name, pf.name, lowest, lowestName)
			// The 256-colour palette is fixed, so it keeps the text floor. The
			// 16 colours are whatever the terminal defines; the xterm defaults
			// used here are only indicative, so ANSI16 gets WCAG's 3:1 floor
			// for large text and UI components.
			floor := minTextContrastRatio
			if pf.p == ansi.ANSI16 {
				floor = 3.0
			}
			if pr.name == "HighContrast" && lowest < floor {
				t.Errorf("HighContrast after ForProfile(%s): %s contrast %.2f, want >= %.1f",
					pf.name, lowestName, lowest, floor)
			}
		}
	}
}
