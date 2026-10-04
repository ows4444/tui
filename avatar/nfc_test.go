package avatar

import (
	"os"
	"strconv"
	"strings"
	"testing"
)

func hexRunes(t *testing.T, s string) string {
	t.Helper()
	var b strings.Builder
	for _, f := range strings.Fields(s) {
		v, err := strconv.ParseUint(f, 16, 32)
		if err != nil {
			t.Fatalf("bad code point %q", f)
		}
		b.WriteRune(rune(v))
	}
	return b.String()
}

// conformance checks nfc against the lines of a NormalizationTest file: for
// columns c1..c5, c2 is the NFC of c1, c2 and c3, and c4 is the NFC of c4
// and c5. It returns how many lines it checked.
func conformance(t *testing.T, data string) int {
	t.Helper()
	n := 0
	for _, line := range strings.Split(data, "\n") {
		if i := strings.IndexByte(line, '#'); i >= 0 {
			line = line[:i]
		}
		if line = strings.TrimSpace(line); line == "" || strings.HasPrefix(line, "@") {
			continue
		}
		cols := strings.Split(strings.TrimSuffix(line, ";"), ";")
		if len(cols) != 5 {
			t.Fatalf("%d columns in %q", len(cols), line)
		}
		var c [5]string
		for i := range c {
			c[i] = hexRunes(t, cols[i])
		}
		for i, want := range [5]string{c[1], c[1], c[1], c[3], c[3]} {
			if got := nfc(c[i]); got != want {
				t.Errorf("%s: nfc(column %d) = %+q, want %+q", line, i+1, got, want)
			}
		}
		n++
	}
	return n
}

// nfc passes the checked-in sample of the Unicode conformance test.
func TestNFCPassesTheConformanceSample(t *testing.T) {
	data, err := os.ReadFile("testdata/nfc_vectors.txt")
	if err != nil {
		t.Fatal(err)
	}
	if n := conformance(t, string(data)); n < 3000 {
		t.Errorf("only %d conformance lines checked", n)
	}
}

func TestNFCCases(t *testing.T) {
	for in, want := range map[string]string{
		"":               "",
		"alain":          "alain",
		"café":           "café",
		"café":          "café",
		"Å":              "Å",          // the angstrom sign is a singleton
		"Å":             "Å",          // A and a ring above
		"q̣̇":            "q̣̇",        // marks are put in order
		"ẛ̣":             "ẛ̣",         // long s with dot above, then dot below
		"가":             "가",          // Hangul L and V
		"각":            "각",          // Hangul L, V and T
		"각":             "각",          // Hangul LV and T
		"각":              "각",          // Hangul LVT is stable
		"क़":              "क़",         // an excluded composite stays apart
		"ǟ":            "ǟ",          // two marks compose in turn
		"Σ́":             "Σ́",         // nothing to compose to
		"\U0001F98A":     "\U0001F98A", // outside the BMP, untouched
		"é\U0001F98Aé": "é\U0001F98Aé",
		"́":              "́",  // a mark with no starter
		"̈́":              "̈́", // decomposes and has no starter to join
	} {
		if got := nfc(in); got != want {
			t.Errorf("nfc(%+q) = %+q, want %+q", in, got, want)
		}
	}
}

// The tables are sorted, as the lookups assume.
func TestNFCTablesAreSorted(t *testing.T) {
	for i := 1; i < len(nfcClasses); i++ {
		if nfcClasses[i].lo <= nfcClasses[i-1].hi {
			t.Fatalf("nfcClasses is not sorted at %d", i)
		}
	}
	for i := 1; i < len(nfcDecomps); i++ {
		if nfcDecomps[i][0] <= nfcDecomps[i-1][0] {
			t.Fatalf("nfcDecomps is not sorted at %d", i)
		}
	}
	for i := 1; i < len(nfcComposites); i++ {
		a, b := nfcComposites[i-1], nfcComposites[i]
		if b[0] < a[0] || b[0] == a[0] && b[1] <= a[1] {
			t.Fatalf("nfcComposites is not sorted at %d", i)
		}
	}
}
