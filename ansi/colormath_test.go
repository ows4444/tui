package ansi

import (
	"math"
	"testing"
)

func TestHSLKnownColours(t *testing.T) {
	for _, tt := range []struct {
		c       RGB
		h, s, l float64
	}{
		{RGB{255, 0, 0}, 0, 1, 0.5},
		{RGB{0, 255, 0}, 120, 1, 0.5},
		{RGB{0, 0, 255}, 240, 1, 0.5},
		{RGB{255, 255, 255}, 0, 0, 1},
		{RGB{0, 0, 0}, 0, 0, 0},
		{RGB{128, 128, 128}, 0, 0, 128.0 / 255},
		{RGB{255, 0, 128}, 330.1, 1, 0.5},
	} {
		h, s, l := tt.c.HSL()
		if math.Abs(h-tt.h) > 0.5 || math.Abs(s-tt.s) > 0.01 || math.Abs(l-tt.l) > 0.01 {
			t.Errorf("%v.HSL() = %.1f, %.2f, %.2f; want %.1f, %.2f, %.2f", tt.c, h, s, l, tt.h, tt.s, tt.l)
		}
	}
}

func TestFromHSLRoundTrips(t *testing.T) {
	for r := 0; r < 256; r += 51 {
		for g := 0; g < 256; g += 51 {
			for b := 0; b < 256; b += 51 {
				c := RGB{uint8(r), uint8(g), uint8(b)}
				h, s, l := c.HSL()
				if got := FromHSL(h, s, l); got != c {
					t.Errorf("FromHSL(%v.HSL()) = %v", c, got)
				}
			}
		}
	}
}

func TestFromHSLWrapsAndClamps(t *testing.T) {
	if FromHSL(-120, 1, 0.5) != FromHSL(240, 1, 0.5) || FromHSL(480, 1, 0.5) != FromHSL(120, 1, 0.5) {
		t.Fatal("hue does not wrap")
	}
	if FromHSL(10, 5, 2) != (RGB{255, 255, 255}) || FromHSL(10, -1, -1) != (RGB{}) {
		t.Fatal("saturation and lightness are not clamped")
	}
}

func TestComplement(t *testing.T) {
	if got := (RGB{255, 0, 0}).Complement(); got != (RGB{0, 255, 255}) {
		t.Fatalf("complement of red = %v", got)
	}
	if got := (RGB{90, 90, 90}).Complement(); got != (RGB{90, 90, 90}) {
		t.Fatalf("a grey has no opposite: %v", got)
	}
}

func TestLuminanceAndReadable(t *testing.T) {
	if (RGB{}).Luminance() != 0 || math.Abs((RGB{255, 255, 255}).Luminance()-1) > 1e-9 {
		t.Fatal("black and white are 0 and 1")
	}
	for _, tt := range []struct{ bg, want RGB }{
		{RGB{}, RGB{255, 255, 255}},
		{RGB{255, 255, 255}, RGB{}},
		{RGB{30, 30, 30}, RGB{255, 255, 255}},
		{RGB{250, 250, 200}, RGB{}},
		{RGB{0, 0, 200}, RGB{255, 255, 255}},
		{RGB{255, 255, 0}, RGB{}},
	} {
		if got := tt.bg.Readable(); got != tt.want {
			t.Errorf("%v.Readable() = %v; want %v", tt.bg, got, tt.want)
		}
	}
}
