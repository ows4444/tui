package ansi

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The generated table must be sorted and non-overlapping, or binary search
// misses runes. Ranges of equal width that touch are merged by the generator.
func TestGeneratedWidthTableIsSortedAndDisjoint(t *testing.T) {
	tbl := generatedWidthRanges
	if len(tbl) == 0 {
		t.Fatal("generatedWidthRanges is empty")
	}
	for i, r := range tbl {
		if r.lo > r.hi || r.hi > 0x10FFFF {
			t.Errorf("[%d] = %#x..%#x is not a valid range", i, r.lo, r.hi)
		}
		if r.w != 0 && r.w != 2 {
			t.Errorf("[%d] = %#x..%#x has width %d, want 0 or 2", i, r.lo, r.hi, r.w)
		}
		if i == 0 {
			continue
		}
		prev := tbl[i-1]
		if r.lo <= prev.hi {
			t.Errorf("[%d] = %#x..%#x overlaps or is unsorted after %#x..%#x", i, r.lo, r.hi, prev.lo, prev.hi)
		}
		if r.lo == prev.hi+1 && r.w == prev.w {
			t.Errorf("[%d] = %#x..%#x touches %#x..%#x with the same width; the generator should have merged them", i, r.lo, r.hi, prev.lo, prev.hi)
		}
	}
}

func TestUnicodeVersionIsRecorded(t *testing.T) {
	if !regexp.MustCompile(`^\d+\.\d+\.\d+$`).MatchString(unicodeVersion) {
		t.Errorf("unicodeVersion = %q, want major.minor.patch", unicodeVersion)
	}
}

// Spot checks against the pinned data, one from each class and one gap.
func TestRuneWidthKnownRunes(t *testing.T) {
	cases := []struct {
		r    rune
		want int
	}{
		{0x0301, 0},  // combining acute accent: Mn
		{0x200D, 0},  // zero width joiner: Cf
		{0x3099, 0},  // combining kana mark: Mn, although East Asian Wide
		{0x4E2D, 2},  // CJK ideograph: W
		{0xFF21, 2},  // fullwidth A: F
		{0x1F600, 2}, // grinning face: Emoji_Presentation
		{0x1F1E6, 2}, // regional indicator A: Emoji_Presentation
		{'A', 1},     // narrow
		{0x00E9, 1},  // é
		{0x2603, 1},  // snowman: Emoji but text presentation
	}
	for _, c := range cases {
		if got := runeWidth(c.r); got != c.want {
			t.Errorf("runeWidth(U+%04X) = %d, want %d", c.r, got, c.want)
		}
	}
}

// runeWidth (binary search) must agree with a sequential sweep of the table
// for every rune, so no rune is lost between the searched ranges.
func TestRuneWidthMatchesASweepOfTheTableForEveryRune(t *testing.T) {
	next := 0 // first table row that ends at or after r
	for r := rune(0x300); r <= 0x10FFFF; r++ {
		for next < len(generatedWidthRanges) && generatedWidthRanges[next].hi < r {
			next++
		}
		want := 1
		if next < len(generatedWidthRanges) && generatedWidthRanges[next].lo <= r {
			want = int(generatedWidthRanges[next].w)
		}
		if got := runeWidth(r); got != want {
			t.Fatalf("runeWidth(U+%04X) = %d, want %d", r, got, want)
		}
	}
}

// The columns for runes whose width changed with the Unicode 17.0.0 tables.
func TestWidthOfRunesThatChangedWithTheGeneratedTables(t *testing.T) {
	cases := []struct {
		s    string
		want int
		why  string
	}{
		{"\u4DC0", 2, "hexagram symbol: wide, missing from the old table"},
		{"\U0001D300", 2, "Tai Xuan Jing monogram: wide"},
		{"\U00018B00", 2, "Khitan Small Script: wide"},
		{"\u0F99", 0, "Tibetan subjoined letter: Mn, missing from the old table"},
		{"\u2DE0", 0, "combining Cyrillic letter: Mn"},
		{"\U000E0020", 0, "tag space: Cf"},
		{"\U0001F266", 1, "unassigned, inside a block the old table widened"},
		{"\u3248", 1, "East Asian Ambiguous, not Wide"},
		{"\u3099", 0, "combining kana mark: Mn even though East Asian Wide"},
		{"\u1ADE", 1, "listed as zero width before, but not Mn/Me/Cf"},
		{"a\u0301", 1, "letter plus combining accent"},
		{"\u00AD", 1, "soft hyphen stays one column"},
		{"\u4E2D\u6587", 4, "two CJK ideographs"},
	}
	for _, c := range cases {
		if got := Width(c.s); got != c.want {
			t.Errorf("Width(%q) = %d, want %d (%s)", c.s, got, c.want, c.why)
		}
	}
}

// The package doc must name the Unicode version the tables come from, so it
// cannot drift when the tables are regenerated.
func TestPackageDocStatesTheUnicodeVersion(t *testing.T) {
	files := parseSources(t, ".", parser.ParseComments)
	doc := ""
	for _, f := range files {
		if f.Doc != nil {
			doc += f.Doc.Text()
		}
	}
	if !strings.Contains(doc, "Unicode "+unicodeVersion) {
		t.Errorf("the package doc does not mention %q:\n%s", "Unicode "+unicodeVersion, doc)
	}
}

// parseSources parses the non-test Go files in dir. It replaces
// parser.ParseDir, deprecated since Go 1.25; these checks read declarations
// and comments only, so the build tags ParseDir ignored do not matter here.
func parseSources(t *testing.T, dir string, mode parser.Mode) []*ast.File {
	t.Helper()
	names, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	var files []*ast.File
	for _, name := range names {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, mode)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, f)
	}
	return files
}
