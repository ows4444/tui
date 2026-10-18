package normname

import (
	"strings"
	"testing"
	"time"
	"unicode"
	"unicode/utf8"
)

// bom is U+FEFF, written as bytes so no tool rewrites it.
const bom = "\xef\xbb\xbf"

func TestNameNormalizesTrimsAndLowercases(t *testing.T) {
	for in, want := range map[string]string{
		"Alain":           "alain",
		"  ALAIN\t":       "alain",
		" Ada ":           "ada",
		bom + "bob" + bom: "bob",
		"\u0085x\u0085":   "\u0085x\u0085", // U+0085 is not trimmed
		"Café":           "café",
		"CAFÉ":            "café",
		"İstanbul":        "i̇stanbul",
		"ΟΔΥΣΣΕΥΣ":        "οδυσσευς", // final sigma
		"Σ":               "σ",        // a lone sigma is not final
		"":                "",
		"   ":             "",
	} {
		if got := Name(in); got != want {
			t.Errorf("Name(%+q) = %+q, want %+q", in, got, want)
		}
	}
}

func TestNameIsCutAtARuneBoundary(t *testing.T) {
	long := strings.Repeat("a", 1<<20)
	if got := Name(long); len(got) != MaxName {
		t.Fatalf("len %d, want %d", len(got), MaxName)
	}
	// 3-byte runes: the cut lands inside one and must back up
	three := strings.Repeat("€", 1000)
	got := Name(three)
	if !utf8.ValidString(got) || len(got) > MaxName || len(got) < MaxName-3 {
		t.Fatalf("len %d valid %v", len(got), utf8.ValidString(got))
	}
	if Name(long) != Name(long+"b") {
		t.Fatal("what comes after the cut changed the name")
	}
	if Name(strings.Repeat("a", MaxName-1)+"b") == Name(strings.Repeat("a", MaxName)) {
		t.Fatal("the last kept byte was dropped")
	}
}

func TestALongNameCostsNoMoreThanAShortOne(t *testing.T) {
	long := strings.Repeat("é", 5<<20)
	start := time.Now()
	_ = Name(long)
	if d := time.Since(start); d > 50*time.Millisecond {
		t.Fatalf("10 MB name took %v", d)
	}
}

// The tables are frozen at UnicodeVersion; while Go's own are the same version
// they must agree with them, which proves the generator reads the right data.
// Once Go moves on they may differ, and that is the point of having them.
func TestTablesAgreeWithGoWhileTheVersionsMatch(t *testing.T) {
	if unicode.Version != UnicodeVersion {
		t.Skipf("Go has Unicode %s, the tables %s: they may differ now", unicode.Version, UnicodeVersion)
	}
	for r := rune(0); r <= unicode.MaxRune; r++ {
		if got, want := lowerRune(r), unicode.ToLower(r); got != want {
			t.Fatalf("lowerRune(U+%04X) = U+%04X, Go says U+%04X", r, got, want)
		}
		if got, want := inRuns(spaceRuns[:], r), unicode.Is(unicode.Zs, r); got != want {
			t.Fatalf("space U+%04X: %v, Go says %v", r, got, want)
		}
		cased := unicode.IsUpper(r) || unicode.IsLower(r) || unicode.IsTitle(r) || unicode.Is(unicode.Other_Lowercase, r) || unicode.Is(unicode.Other_Uppercase, r)
		if got := inRuns(casedRuns[:], r); got != cased {
			t.Fatalf("cased U+%04X: %v, Go says %v", r, got, cased)
		}
	}
}

func TestTablesAreSortedAndDisjoint(t *testing.T) {
	for name, runs := range map[string][][2]rune{"space": spaceRuns[:], "cased": casedRuns[:], "ignorable": ignorableRuns[:]} {
		for i, r := range runs {
			if r[0] > r[1] || i > 0 && runs[i-1][1] >= r[0] {
				t.Errorf("%s run %d is out of order: %v after %v", name, i, r, runs[i-1])
			}
		}
	}
	for i := 1; i < len(lowerMap); i++ {
		if lowerMap[i-1][0] >= lowerMap[i][0] {
			t.Fatalf("lowerMap is out of order at %d", i)
		}
	}
}

func TestCaseIgnorableRunesDoNotBreakAFinalSigma(t *testing.T) {
	// a cased letter, an apostrophe (ignorable), the sigma, then a full stop
	if got := Name("A'Σ."); got != "a'ς." {
		t.Fatalf("%+q", got)
	}
	// a cased letter follows: not final
	if got := Name("ΣA"); got != "σa" {
		t.Fatalf("%+q", got)
	}
}
