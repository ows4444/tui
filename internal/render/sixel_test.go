package render

import "testing"

func TestIsSixel(t *testing.T) {
	for seq, want := range map[string]bool{
		"\x1bPq#0;2;0;0;0\x1b\\":     true,
		"\x1bP0;1;0q#0\x1b\\":        true,
		"\x1bP9q\x1b\\":              true,
		"\x1bPtmux;\x1b\x1b_G\x1b\\": false,
		"\x1bP$q\"p\x1b\\":           false, // DECRQSS, not Sixel
		"\x1b_Ga=T;AAAA\x1b\\":       false,
		"\x1bP":                      false,
		"":                           false,
	} {
		if got := isSixel(seq); got != want {
			t.Errorf("isSixel(%q) = %v, want %v", seq, got, want)
		}
	}
}
