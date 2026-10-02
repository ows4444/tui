package ansi

import (
	"math/rand"
	"testing"
)

// refWidth is the original allocating Width, kept verbatim as the reference
// the allocation-free implementation must match on every input.
func refWidth(s string) int {
	w := 0
	for _, r := range StripANSI(s) {
		w += runeWidth(r)
	}
	return w
}

var widthPieces = []string{
	"", "a", "hello", " ", "\t", "\n", "你好", "🎉", "é", "é", "\u200b", "│", "─",
	"\x1b[1m", "\x1b[0m", "\x1b[38;2;1;2;3m", "\x1b[", "\x1b", "\x1b]8;;u\x07", "\x1b[Z",
	// Fragments of multi-byte runes, so escapes can land inside one: the old
	// implementation removes the escape and re-joins the bytes.
	"\xe2", "\x82", "\xac", "\xe2\x82", "\xf0\x9f", "\x8e\x89", "\xff", "\x80",
}

func TestWidthMatchesReferenceOnRandomInputs(t *testing.T) {
	r := rand.New(rand.NewSource(7))
	for i := 0; i < 300000; i++ {
		var s string
		for j, n := 0, r.Intn(8); j < n; j++ {
			s += widthPieces[r.Intn(len(widthPieces))]
		}
		if got, want := Width(s), refWidth(s); got != want {
			t.Fatalf("Width(%q) = %d, want %d", s, got, want)
		}
	}
}

func TestWidthSplitRuneAcrossAnEscapeMatchesReference(t *testing.T) {
	// "€" is E2 82 AC; an escape sequence between its bytes is removed by
	// StripANSI, which re-joins them into one wide-or-narrow rune.
	for _, s := range []string{"\xe2\x1b[1m\x82\xac", "\xe2\x82\x1b[m\xac", "a\xf0\x9f\x1b[0m\x8e\x89b"} {
		if got, want := Width(s), refWidth(s); got != want {
			t.Errorf("Width(%q) = %d, want %d", s, got, want)
		}
	}
}

func FuzzWidthMatchesReference(f *testing.F) {
	for _, p := range widthPieces {
		f.Add(p)
	}
	f.Add("\xe2\x1b[1m\x82\xac")
	f.Add("\x1b[38;2;255;0;0mred\x1b[0m 你好")
	// The reference counts rune by rune, so compare with clusters off.
	SetClusterWidth(false)
	f.Cleanup(func() { SetClusterWidth(true) })
	f.Fuzz(func(t *testing.T, s string) {
		if got, want := Width(s), refWidth(s); got != want {
			t.Fatalf("Width(%q) = %d, want %d", s, got, want)
		}
	})
}
