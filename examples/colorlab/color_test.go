package main

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
)

func TestParseHex(t *testing.T) {
	good := map[string]ansi.RGB{
		"#3A7BD5": {R: 0x3A, G: 0x7B, B: 0xD5}, "3a7bd5": {R: 0x3A, G: 0x7B, B: 0xD5},
		"#fff": {R: 255, G: 255, B: 255}, " #000 ": {}, "#0a0": {R: 0, G: 0xAA, B: 0},
	}
	for in, want := range good {
		if got, ok := parseHex(in); !ok || got != want {
			t.Errorf("parseHex(%q) = %v, %v; want %v", in, got, ok, want)
		}
	}
	for _, bad := range []string{"", "#", "#12", "#12345", "#1234567", "#ggg", "red", "#12 456"} {
		if _, ok := parseHex(bad); ok {
			t.Errorf("parseHex(%q) accepted", bad)
		}
	}
}

func TestHexRoundTrip(t *testing.T) {
	for _, c := range []ansi.RGB{{}, {R: 255, G: 255, B: 255}, {R: 1, G: 2, B: 3}, {R: 0x3A, G: 0x7B, B: 0xD5}} {
		if got, ok := parseHex(hex(c)); !ok || got != c {
			t.Errorf("%v -> %s -> %v", c, hex(c), got)
		}
	}
}

func TestGradeThresholds(t *testing.T) {
	for r, want := range map[float64]string{21: "AAA", 7: "AAA", 6.99: "AA", 4.5: "AA", 4.49: "AA large", 3: "AA large", 2.99: "fail", 1: "fail"} {
		if got := grade(r); got != want {
			t.Errorf("grade(%v) = %q, want %q", r, got, want)
		}
	}
}

func TestContrastKnownValues(t *testing.T) {
	if r := contrast(black, white); r < 20.9 || r > 21.1 {
		t.Errorf("black on white = %v, want 21", r)
	}
	if r := contrast(white, white); r < 0.99 || r > 1.01 {
		t.Errorf("white on white = %v, want 1", r)
	}
}

func TestPaletteAndCSS(t *testing.T) {
	c := ansi.RGB{R: 0x3A, G: 0x7B, B: 0xD5}
	ps := palette(c)
	if len(ps) != 6 || ps[2].c != c || ps[2].name != "base" {
		t.Fatalf("palette = %v", ps)
	}
	if contrast(ps[0].c, white) <= contrast(ps[4].c, white) {
		t.Error("the darkest shade is not more contrasting with white than the lightest")
	}
	out := css(ps)
	for _, want := range []string{":root {", "--color-base: #3a7bd5;", "--color-complement:", "--color-dark-2:", "}"} {
		if !strings.Contains(out, want) {
			t.Errorf("css lacks %q:\n%s", want, out)
		}
	}
	if strings.Count(out, "\n") != 8 {
		t.Errorf("css has %d lines, want 8:\n%s", strings.Count(out, "\n"), out)
	}
}
