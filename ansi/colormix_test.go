package ansi

import (
	"fmt"
	"testing"
)

func TestMixEndsAndMiddle(t *testing.T) {
	a, b := RGB{R: 0, G: 100, B: 200}, RGB{R: 200, G: 100, B: 0}
	for _, tc := range []struct {
		name string
		t    float64
		want RGB
	}{
		{"zero is the receiver", 0, a},
		{"one is the other", 1, b},
		{"half", 0.5, RGB{R: 100, G: 100, B: 100}},
		{"below zero clamps", -3, a},
		{"above one clamps", 7, b},
	} {
		if got := a.Mix(b, tc.t); got != tc.want {
			t.Errorf("%s: Mix(%v) = %+v, want %+v", tc.name, tc.t, got, tc.want)
		}
	}
}

func TestLightenAndDarken(t *testing.T) {
	c := RGB{R: 100, G: 50, B: 0}
	if got := c.Lighten(0); got != c {
		t.Errorf("Lighten(0) = %+v, want %+v", got, c)
	}
	if got := c.Lighten(1); got != (RGB{R: 255, G: 255, B: 255}) {
		t.Errorf("Lighten(1) = %+v, want white", got)
	}
	if got := c.Darken(1); got != (RGB{}) {
		t.Errorf("Darken(1) = %+v, want black", got)
	}
	if got := c.Darken(0.5); got != (RGB{R: 50, G: 25, B: 0}) {
		t.Errorf("Darken(0.5) = %+v", got)
	}
}

func TestBlend(t *testing.T) {
	red, blue := RGB{R: 255}, RGB{B: 255}
	if got := Blend(0, red, blue); got != nil {
		t.Errorf("n = 0: %v, want nil", got)
	}
	if got := Blend(3); got != nil {
		t.Errorf("no stops: %v, want nil", got)
	}
	if got := Blend(1, red, blue); len(got) != 1 || got[0] != red {
		t.Errorf("n = 1: %v, want the first stop", got)
	}
	if got := Blend(3, red); len(got) != 3 || got[0] != red || got[2] != red {
		t.Errorf("one stop: %v, want three copies", got)
	}
	got := Blend(5, red, blue)
	if len(got) != 5 || got[0] != red || got[4] != blue || got[2] != (RGB{R: 128, B: 128}) {
		t.Errorf("two stops over 5: %v", got)
	}
	// Three stops: the middle one lands in the middle cell.
	green := RGB{G: 255}
	got = Blend(5, red, green, blue)
	if got[0] != red || got[2] != green || got[4] != blue {
		t.Errorf("three stops over 5: %v, want the stops at 0, 2 and 4", got)
	}
}

func ExampleRGB_Mix() {
	white := RGB{R: 255, G: 255, B: 255}
	fmt.Println(RGB{}.Mix(white, 0.5))
	// Output: {128 128 128}
}

func ExampleBlend() {
	fmt.Println(Blend(3, RGB{R: 255}, RGB{B: 255}))
	// Output: [{255 0 0} {128 0 128} {0 0 255}]
}
