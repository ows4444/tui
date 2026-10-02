package ansi

import (
	"sort"
	"testing"
)

func TestGraphemeBreakOfKnownRunes(t *testing.T) {
	for _, tc := range []struct {
		r    rune
		want gbClass
	}{
		{'a', gbOther}, {'\r', gbCR}, {'\n', gbLF}, {0x01, gbControl},
		{0x0301, gbExtend}, {0x200D, gbZWJ}, {0x1F1E6, gbRegionalIndicator},
		{0x0600, gbPrepend}, {0x0903, gbSpacingMark},
		{0x1100, gbL}, {0x1160, gbV}, {0x11A8, gbT}, {0xAC00, gbLV}, {0xAC01, gbLVT},
		{0x1F3FB, gbExtend}, // an emoji skin-tone modifier extends
	} {
		if got := graphemeBreakOf(tc.r); got != tc.want {
			t.Errorf("graphemeBreakOf(%U) = %d, want %d", tc.r, got, tc.want)
		}
	}
}

func TestExtPictAndIndicConjunctBreak(t *testing.T) {
	for r, want := range map[rune]bool{'a': false, 0x00A9: true, 0x1F468: true, 0x1F600: true, 0x1F1E6: false} {
		if got := isExtPict(r); got != want {
			t.Errorf("isExtPict(%U) = %v, want %v", r, got, want)
		}
	}
	for r, want := range map[rune]incbClass{'a': incbNone, 0x094D: incbLinker, 0x0915: incbConsonant, 0x0300: incbExtend} {
		if got := incbOf(r); got != want {
			t.Errorf("incbOf(%U) = %d, want %d", r, got, want)
		}
	}
}

// The generated tables are sorted and non-overlapping, so the binary search is
// valid.
func TestGeneratedGraphemeTablesAreSortedAndDisjoint(t *testing.T) {
	check := func(name string, n int, at func(i int) (lo, hi rune)) {
		if !sort.SliceIsSorted(make([]struct{}, n), func(i, j int) bool { lo1, _ := at(i); lo2, _ := at(j); return lo1 < lo2 }) {
			t.Errorf("%s is not sorted", name)
		}
		for i := 0; i < n; i++ {
			lo, hi := at(i)
			if hi < lo {
				t.Fatalf("%s[%d] is reversed", name, i)
			}
			if i > 0 {
				if _, prev := at(i - 1); lo <= prev {
					t.Fatalf("%s[%d] overlaps its predecessor", name, i)
				}
			}
		}
	}
	check("generatedGraphemeBreak", len(generatedGraphemeBreak), func(i int) (rune, rune) {
		return generatedGraphemeBreak[i].lo, generatedGraphemeBreak[i].hi
	})
	check("generatedExtPict", len(generatedExtPict), func(i int) (rune, rune) { return generatedExtPict[i].lo, generatedExtPict[i].hi })
	check("generatedIndicConjunctBreak", len(generatedIndicConjunctBreak), func(i int) (rune, rune) {
		return generatedIndicConjunctBreak[i].lo, generatedIndicConjunctBreak[i].hi
	})
}
