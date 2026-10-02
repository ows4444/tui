package ansi

import (
	"math/rand"
	"os"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"
)

// The segmenter agrees with every case of Unicode's own GraphemeBreakTest.txt
// (Unicode 17.0.0, testdata/GraphemeBreakTest.txt).
func TestSegmenterMatchesTheUnicodeConformanceFile(t *testing.T) {
	data, err := os.ReadFile("testdata/GraphemeBreakTest.txt")
	if err != nil {
		t.Fatal(err)
	}
	cases := 0
	for n, line := range strings.Split(string(data), "\n") {
		if i := strings.IndexByte(line, '#'); i >= 0 {
			line = line[:i]
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		var text strings.Builder
		var want []string
		var cur strings.Builder
		for _, f := range fields {
			switch f {
			case "÷":
				if cur.Len() > 0 {
					want = append(want, cur.String())
					cur.Reset()
				}
			case "×":
			default:
				cp, err := strconv.ParseUint(f, 16, 32)
				if err != nil {
					t.Fatalf("line %d: %v", n+1, err)
				}
				text.WriteRune(rune(cp))
				cur.WriteRune(rune(cp))
			}
		}
		got := splitClusters(text.String())
		if strings.Join(got, "|") != strings.Join(want, "|") {
			t.Errorf("line %d: %q split as %q, want %q", n+1, text.String(), got, want)
		}
		cases++
	}
	if cases < 700 {
		t.Fatalf("only %d cases read", cases)
	}
}

func TestClusterWidths(t *testing.T) {
	for _, tc := range []struct {
		name, s string
		want    int
	}{
		{"family ZWJ sequence", "👨‍👩‍👧‍👦", 2},
		{"flag", "🇩🇪", 2},
		{"two flags", "🇩🇪🇫🇷", 4},
		{"three regional indicators", "🇩🇪🇫", 4},
		{"skin tone", "👋🏽", 2},
		{"heart with VS16", "❤️", 2},
		{"heart without VS16", "❤", 1},
		{"combining accent", "é", 1},
		{"stacked marks", "á̂̃", 1},
		{"hangul jamo syllable", "각", 2},
		{"devanagari conjunct", "क्ष", 1},
		{"copyright with VS16", "©️", 2},
		{"text around a cluster", "a👨‍👩b", 4},
		{"escapes around a cluster", "\x1b[1m👨‍👩\x1b[0m", 2},
		{"escape inside a cluster", "👋\x1b[1m🏽", 2},
		{"ASCII", "hello world", 11},
		{"CJK", "你好", 4},
		{"CR LF measures zero (controls take no column)", "\r\n", 0},
		{"empty", "", 0},
	} {
		if got := Width(tc.s); got != tc.want {
			t.Errorf("%s: Width(%q) = %d, want %d", tc.name, tc.s, got, tc.want)
		}
		if got, want := Width(tc.s), Width(StripANSI(tc.s)); got != want {
			t.Errorf("%s: Width differs from Width(StripANSI): %d vs %d", tc.name, got, want)
		}
	}
}

func TestSetClusterWidthOffCountsEachRune(t *testing.T) {
	t.Cleanup(func() { SetClusterWidth(true) })
	SetClusterWidth(false)
	for s, want := range map[string]int{
		"👨‍👩‍👧‍👦": 8, // four wide runes, three zero-width joiners
		"🇩🇪":      4,
		"👋🏽":      4,
		"é":      1,
		"hello":   5,
	} {
		if got := Width(s); got != want {
			t.Errorf("clusters off: Width(%q) = %d, want %d", s, got, want)
		}
	}
	if got := Truncate("👋🏽", 2); got != "👋" {
		t.Errorf("clusters off: Truncate = %q, want the base only", got)
	}
	SetClusterWidth(true)
	if got := Truncate("👋🏽", 2); got != "👋🏽" {
		t.Errorf("clusters on: Truncate = %q, want the whole cluster", got)
	}
}

func TestTruncateNeverSplitsACluster(t *testing.T) {
	for _, s := range []string{"👨‍👩‍👧‍👦x", "a🇩🇪🇫🇷b", "👋🏽👋🏽", "éé", "❤️❤️", "\x1b[31m👨‍👩\x1b[0mz"} {
		plain := StripANSI(s)
		for w := 0; w <= Width(s)+1; w++ {
			got := StripANSI(Truncate(s, w))
			if Width(got) > w {
				t.Errorf("Truncate(%q, %d) = %q is %d wide", s, w, got, Width(got))
			}
			if !strings.HasPrefix(plain, got) {
				t.Errorf("Truncate(%q, %d) = %q is not a prefix", s, w, got)
			}
			// The cut must fall on a cluster boundary.
			n := 0
			ok := got == ""
			for _, c := range splitClusters(plain) {
				n += len(c)
				if n == len(got) {
					ok = true
				}
			}
			if !ok {
				t.Errorf("Truncate(%q, %d) = %q cuts inside a cluster", s, w, got)
			}
		}
	}
}

func TestTrimLeftWidthKeepsAClusterWhole(t *testing.T) {
	s := "👨‍👩‍👧x"
	if got := TrimLeftWidth(s, 1); got != " x" {
		t.Errorf("TrimLeftWidth(%q, 1) = %q, want the cluster dropped whole with a pad space", s, got)
	}
	if got := TrimLeftWidth(s, 2); got != "x" {
		t.Errorf("TrimLeftWidth(%q, 2) = %q, want x", s, got)
	}
	if got := TrimLeftWidth("ébc", 1); got != "bc" {
		t.Errorf("a combining mark must go with its base: %q", got)
	}
	// Trim and truncate at the same column reassemble the visible text.
	for _, w := range []int{0, 1, 2, 3, 4, 5} {
		left, right := Truncate("a👋🏽bcd", w), TrimLeftWidth("a👋🏽bcd", w)
		if Width(left)+Width(right) < Width("a👋🏽bcd")-1 {
			t.Errorf("w=%d: %q + %q lose text", w, left, right)
		}
	}
}

// For text with no joining sequences a cluster is a rune, so widths match the
// rune-by-rune sum exactly: nothing changes for ordinary text.
func TestOrdinaryTextMeasuresAsBefore(t *testing.T) {
	r := rand.New(rand.NewSource(11))
	pieces := []string{"a", "hello", " ", "\t", "\n", "\r\n", "你好", "🎉", "é", "​", "│", "\x1b[1m", "\x1b[0m", "ü", "ß", "→"}
	for i := 0; i < 20000; i++ {
		var b strings.Builder
		for j, n := 0, r.Intn(8); j < n; j++ {
			b.WriteString(pieces[r.Intn(len(pieces))])
		}
		s := b.String()
		if got, want := Width(s), refWidth(s); got != want {
			t.Fatalf("Width(%q) = %d, rune sum %d", s, got, want)
		}
	}
}

// Width still equals Width of the stripped text on random text that includes
// joiners, flags and modifiers, and never allocates.
func TestClusterWidthAgreesWithStrippedAndDoesNotAllocate(t *testing.T) {
	r := rand.New(rand.NewSource(5))
	pieces := []string{"a", "e", "́", "‍", "👨", "👩", "🇩", "🇪", "🏽", "👋", "️", "❤", "\x1b[1m", "\x1b[0m", "你", "ᄀ", "ᅡ", "क", "्", "ष", "\r", "\n"}
	for i := 0; i < 30000; i++ {
		var b strings.Builder
		for j, n := 0, r.Intn(10); j < n; j++ {
			b.WriteString(pieces[r.Intn(len(pieces))])
		}
		s := b.String()
		// The contract is Width(s) == plainWidth(StripANSI(s)). Measuring
		// Width(StripANSI(s)) would re-read text the strip left behind as a
		// new escape ("\x1b\x1b[A[" strips to "\x1b[").
		if got, want := Width(s), slowWidth(s); got != want {
			t.Fatalf("Width(%q) = %d, stripped %d", s, got, want)
		}
		if !utf8.ValidString(s) {
			continue
		}
		for _, w := range []int{1, 3, 6} {
			if got := Width(Truncate(s, w)); got > w {
				t.Fatalf("Truncate(%q, %d) is %d wide", s, w, got)
			}
		}
	}
	s := "hello 👨‍👩‍👧 world é"
	if n := testing.AllocsPerRun(100, func() { _ = Width(s) }); n != 0 {
		t.Errorf("Width allocates %v times", n)
	}
}

func FuzzClusterWidth(f *testing.F) {
	for _, s := range []string{"", "a", "👨‍👩‍👧", "🇩🇪🇫", "👋🏽", "é", "❤️", "\x1b[1m👋\x1b[0m🏽", "각", "क्ष", "\r\n"} {
		f.Add(s, 2)
	}
	f.Fuzz(func(t *testing.T, s string, w int) {
		// The contract is Width(s) == plainWidth(StripANSI(s)). Measuring
		// Width(StripANSI(s)) would re-read text the strip left behind as a
		// new escape ("\x1b\x1b[A[" strips to "\x1b[").
		if got, want := Width(s), slowWidth(s); got != want {
			t.Fatalf("Width(%q) = %d, stripped %d", s, got, want)
		}
		if w < 0 || w > 50 || !utf8.ValidString(s) {
			return
		}
		if got := Width(Truncate(s, w)); got > w {
			t.Fatalf("Truncate(%q, %d) is %d wide", s, w, got)
		}
	})
}
