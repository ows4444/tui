package avatar

import (
	"math"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// wcagLum is the WCAG relative luminance of an sRGB colour, worked out here and
// not by the code under test.
func wcagLum(c ansi.RGB) float64 {
	f := func(v uint8) float64 {
		s := float64(v) / 255
		if s <= 0.03928 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	return 0.2126*f(c.R) + 0.7152*f(c.G) + 0.0722*f(c.B)
}

func ratio(a, b ansi.RGB) float64 {
	x, y := wcagLum(a), wcagLum(b)
	return (math.Max(x, y) + 0.05) / (math.Min(x, y) + 0.05)
}

// themes are the pages the sweep draws on: the stock dark and light themes, a
// pure black and a pure white page, and a middle grey.
func sweepThemes() map[string]theme.Theme {
	grey := theme.DarkTheme()
	grey.Background = ansi.RGB{R: 0x80, G: 0x80, B: 0x80}
	black, white := theme.DarkTheme(), theme.LightTheme()
	black.Background, white.Background = ansi.RGB{}, ansi.RGB{R: 255, G: 255, B: 255}
	return map[string]theme.Theme{"dark": theme.DarkTheme(), "light": theme.LightTheme(), "black": black, "white": white, "grey": grey, "nord": theme.NordTheme(), "solarized-light": theme.SolarizedLightTheme()}
}

// On every page, in every tone and hue, the body is at least 3:1 against the
// page (or the plate) and the eyes at least 4.5:1 against the body.
func TestBodyAndEyesKeepTheirContrastOnEveryPage(t *testing.T) {
	worstBody, worstEye := math.Inf(1), math.Inf(1)
	for name, th := range sweepThemes() {
		for _, bg := range []Background{BackgroundNone, BackgroundSquircle} {
			for _, tone := range []Tone{TonePastel, TonePale, ToneMid, ToneDeep, ToneBright, ToneInk} {
				for hue := 1.0; hue < 361; hue += 10 {
					m := Model{Name: "x", Width: 8, Height: 4, Hue: hue, Tone: tone, Theme: th, Background: bg}
					body, eyes, plate := m.Colors()
					ref := ansi.RGB{}
					if bg == BackgroundNone {
						ref = m.pageRGB()
					} else {
						ref = plate
					}
					if c := ratio(body, ref); c < worstBody {
						worstBody = c
						if c < 3.0 {
							t.Errorf("%s bg %d tone %v hue %v: body %.2f:1 against %v", name, bg, tone, hue, c, ref)
						}
					}
					if c := ratio(body, eyes); c < worstEye {
						worstEye = c
						if c < 4.5 {
							t.Errorf("%s bg %d tone %v hue %v: eyes %.2f:1 against the body", name, bg, tone, hue, c)
						}
					}
				}
			}
		}
	}
	t.Logf("worst body %.2f:1, worst eyes %.2f:1", worstBody, worstEye)
}

// An expression's tint keeps both floors, at every heat on the way.
func TestTintedBodyKeepsItsFloorOnEveryPage(t *testing.T) {
	for name, th := range sweepThemes() {
		for _, e := range []Expression{ExpressionMad, ExpressionLove, ExpressionShy, ExpressionSick} {
			for _, bg := range []Background{BackgroundNone, BackgroundCircle} {
				for _, tone := range []Tone{TonePastel, ToneMid, ToneInk} {
					m := Model{Name: "tint", Width: 8, Height: 4, Tone: tone, Theme: th, Background: bg, Expression: e}
					body, eyes, plate := m.Colors()
					ref := plate
					if bg == BackgroundNone {
						ref = m.pageRGB()
					}
					if c := ratio(body, ref); c < 3.0 {
						t.Errorf("%s %v bg %d tone %v: body %.2f:1", name, e, bg, tone, c)
					}
					if c := ratio(body, eyes); c < 4.5 {
						t.Errorf("%s %v bg %d tone %v: eyes %.2f:1", name, e, bg, tone, c)
					}
				}
			}
		}
	}
}

// The same name is a different colour on a light page than on a dark one: the
// theme is used.
func TestTheThemeChangesTheColours(t *testing.T) {
	dark, light := New("ada"), New("ada").SetTheme(theme.LightTheme())
	db, _, _ := dark.Colors()
	lb, _, _ := light.Colors()
	if db == lb {
		t.Fatalf("the body is %v on both pages", db)
	}
	if dark.View() == light.View() {
		t.Fatal("the View does not follow the theme")
	}
	if string(dark.PNG(64)) == string(light.PNG(64)) {
		t.Fatal("the PNG does not follow the theme")
	}
	if dark.SVG() == light.SVG() {
		t.Fatal("the SVG does not follow the theme")
	}
}

// A model with no theme at all is drawn for a dark page.
func TestNoThemeMeansADarkPage(t *testing.T) {
	bare := Model{Name: "ada", Width: 8, Height: 4}
	if got := bare.pageRGB(); got != darkSurface.rgb() {
		t.Fatalf("page %v", got)
	}
	body, _, _ := bare.Colors()
	if c := ratio(body, darkSurface.rgb()); c < 3.0 {
		t.Fatalf("body %.2f:1", c)
	}
}

func TestPageComesFromTheThemesBackgroundThenItsInverseText(t *testing.T) {
	th := theme.DarkTheme()
	th.Background = ansi.RGB{R: 1, G: 2, B: 3}
	if got := (Model{Theme: th}).pageRGB(); got != (ansi.RGB{R: 1, G: 2, B: 3}) {
		t.Fatalf("%v", got)
	}
	th.Background = nil
	th.TextInverse = ansi.Color256(21)
	if got := (Model{Theme: th}).pageRGB(); got != ansi.Color256(21).RGB() {
		t.Fatalf("%v", got)
	}
	th.TextInverse = ansi.BasicColor(4)
	if got := (Model{Theme: th}).pageRGB(); got != ansi.BasicColor(4).RGB() {
		t.Fatalf("%v", got)
	}
	th.TextInverse = nil
	if got := (Model{Theme: th}).pageRGB(); got != darkSurface.rgb() {
		t.Fatalf("%v", got)
	}
}

// The plate is the other end of the lightness scale from the body.
func TestPlateSetsTheBodyOff(t *testing.T) {
	for _, tone := range []Tone{TonePastel, ToneBright, ToneMid, ToneInk} {
		m := Model{Name: "x", Width: 8, Height: 4, Tone: tone, Background: BackgroundSquare, Theme: theme.DarkTheme()}
		body, _, plate := m.Colors()
		if (wcagLum(body) > wcagLum(plate)) == (tone == ToneInk) {
			t.Errorf("tone %v: the body %v and the plate %v are on the wrong sides", tone, body, plate)
		}
	}
}
