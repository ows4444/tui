package ansi

import (
	"math/rand"
	"strings"
	"testing"
)

func invert(c RGB) RGB { return RGB{R: 255 - c.R, G: 255 - c.G, B: 255 - c.B} }

func TestFilterColorsRewritesEveryKindOfColour(t *testing.T) {
	cases := map[string]string{
		"\x1b[31m":                 "\x1b[38;2;50;255;255m",     // 205,0,0 -> inverted
		"\x1b[91m":                 "\x1b[38;2;0;255;255m",      // 255,0,0
		"\x1b[42m":                 "\x1b[48;2;255;50;255m",     // 0,205,0
		"\x1b[102m":                "\x1b[48;2;255;0;255m",      // 0,255,0
		"\x1b[38;5;196m":           "\x1b[38;2;0;255;255m",      // 256-colour red
		"\x1b[48;5;21m":            "\x1b[48;2;255;255;0m",      // 256-colour blue
		"\x1b[38;2;10;20;30m":      "\x1b[38;2;245;235;225m",    // truecolor
		"\x1b[58;2;0;0;0m":         "\x1b[58;2;255;255;255m",    // underline colour
		"\x1b[38:2::10:20:30m":     "\x1b[38;2;245;235;225m",    // colon form
		"\x1b[38:5:196m":           "\x1b[38;2;0;255;255m",      // colon 256
		"\x1b[1;31;4m":             "\x1b[1;38;2;50;255;255;4m", // attributes keep their place
		"\x1b[0m":                  "\x1b[0m",
		"\x1b[m":                   "\x1b[m",
		"\x1b[39;49;59m":           "\x1b[39;49;59m", // defaults are the terminal's own
		"\x1b[1;3;4:3;9m":          "\x1b[1;3;4:3;9m",
		"\x1b[38;5m":               "\x1b[38;5m",   // truncated: left as written
		"\x1b[38;9;1m":             "\x1b[38;9;1m", // unknown colour space
		"\x1b[38;2;1;2m":           "\x1b[38;2;1;2m",
		"\x1b[38;5;999m":           "\x1b[38;5;999m",           // out of range
		"\x1b[1;x;31m":             "\x1b[1;x;31m",             // x ends the sequence; the rest is text
		"\x1b[?25l\x1b[2J\x1b[H":   "\x1b[?25l\x1b[2J\x1b[H",   // not SGR
		"\x1b]8;;http://a.b\x1b\\": "\x1b]8;;http://a.b\x1b\\", // a link
		"plain \x1b[":              "plain \x1b[",              // unterminated
	}
	for in, want := range cases {
		if got := FilterColors(in, invert); got != want {
			t.Errorf("FilterColors(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFilterColorsKeepsTextAndNilIsIdentity(t *testing.T) {
	s := "é日本\x1b[31m語\x1b[0m\n\x1b[1mok\x1b[0m"
	if got := StripANSI(FilterColors(s, invert)); got != StripANSI(s) {
		t.Errorf("text changed: %q", got)
	}
	if FilterColors(s, nil) != s {
		t.Error("a nil filter changed the string")
	}
	if got := FilterColors("no escapes", invert); got != "no escapes" {
		t.Error("a string without escapes changed")
	}
}

func TestFilterColorsIdentityKeepsTheColours(t *testing.T) {
	s := "\x1b[31;1mhi\x1b[38;5;196m!\x1b[0m"
	same := FilterColors(s, func(c RGB) RGB { return c })
	// Same colours, written as truecolor: filtering again changes nothing.
	if FilterColors(same, func(c RGB) RGB { return c }) != same {
		t.Errorf("not stable: %q", same)
	}
	if !strings.Contains(same, "38;2;205;0;0") || !strings.Contains(same, "38;2;255;0;0") {
		t.Errorf("colours changed: %q", same)
	}
}

func TestFilterColorsNeverPanicsOnJunk(t *testing.T) {
	r := rand.New(rand.NewSource(7))
	pieces := []string{"\x1b[", "\x1b", "38;2;1;2;3", "38;5;", "48", ";", ":", "m", "3", "9", "0", "1", "x", "é", "\x1b]8;;", "\x1b\\"}
	for n := 0; n < 5000; n++ {
		var b strings.Builder
		for i, k := 0, r.Intn(14); i < k; i++ {
			b.WriteString(pieces[r.Intn(len(pieces))])
		}
		_ = FilterColors(b.String(), invert)
	}
}

func FuzzFilterColors(f *testing.F) {
	for _, s := range []string{"\x1b[31m", "\x1b[38;2;1;2;3mx", "\x1b[38:2::1:2:3m", "a\x1b[", "\x1b[38;5m"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		out := FilterColors(s, invert)
		if StripANSI(out) != StripANSI(s) && !strings.Contains(s, "\x1b") {
			t.Fatalf("text changed: %q -> %q", s, out)
		}
	})
}

func near(a, b uint8, tol int) bool {
	d := int(a) - int(b)
	return d >= -tol && d <= tol
}

func TestGrayscale(t *testing.T) {
	for _, c := range []RGB{{255, 0, 0}, {0, 255, 0}, {0, 0, 255}, {12, 200, 90}, {255, 255, 255}, {0, 0, 0}} {
		g := Grayscale(c)
		if g.R != g.G || g.G != g.B {
			t.Errorf("Grayscale(%v) = %v, not gray", c, g)
		}
	}
	if Grayscale(RGB{255, 255, 255}) != (RGB{255, 255, 255}) || Grayscale(RGB{}) != (RGB{}) {
		t.Error("black and white moved")
	}
	// Green is brighter than red, red than blue.
	r, g, b := Grayscale(RGB{R: 255}).R, Grayscale(RGB{G: 255}).R, Grayscale(RGB{B: 255}).R
	if !(g > r && r > b) {
		t.Errorf("luminance order: red %d green %d blue %d", r, g, b)
	}
}

func TestColourBlindnessSimulationsKeepGraysAndChangeColours(t *testing.T) {
	for name, f := range map[string]func(RGB) RGB{"protan": Protanopia, "deutan": Deuteranopia, "tritan": Tritanopia} {
		for _, v := range []uint8{0, 40, 128, 200, 255} {
			c := RGB{v, v, v}
			if got := f(c); !near(got.R, v, 1) || !near(got.G, v, 1) || !near(got.B, v, 1) {
				t.Errorf("%s moved the gray %d to %v", name, v, got)
			}
		}
	}
	if red := Protanopia(RGB{R: 255}); red.R > 120 {
		t.Errorf("Protanopia(red) = %v, want a dark colour", red)
	}
	if got := Deuteranopia(RGB{G: 255}); got.G > 235 || got == (RGB{G: 255}) {
		t.Errorf("Deuteranopia(green) = %v, want it changed", got)
	}
	if got := Tritanopia(RGB{B: 255}); got == (RGB{B: 255}) {
		t.Errorf("Tritanopia(blue) = %v, want it changed", got)
	}
}

func TestDim(t *testing.T) {
	if got := Dim(1)(RGB{100, 200, 50}); got != (RGB{100, 200, 50}) {
		t.Errorf("Dim(1) = %v", got)
	}
	if got := Dim(0)(RGB{100, 200, 50}); got != (RGB{}) {
		t.Errorf("Dim(0) = %v", got)
	}
	if got := Dim(0.5)(RGB{100, 200, 50}); got != (RGB{50, 100, 25}) {
		t.Errorf("Dim(0.5) = %v", got)
	}
	if Dim(-3)(RGB{9, 9, 9}) != (RGB{}) || Dim(7)(RGB{9, 9, 9}) != (RGB{9, 9, 9}) {
		t.Error("Dim did not clamp")
	}
}
