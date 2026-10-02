package bidi

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"testing"
)

// runConformance checks Resolve against a BidiCharacterTest.txt file: each data
// line gives code points, a paragraph direction (0 left to right, 1 right to
// left, 2 auto), the resolved paragraph level, the resolved level of each
// character ("x" for one X9 removes) and the visual order of the kept ones.
func runConformance(t *testing.T, path string) (lines int) {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Skipf("no conformance data: %v", err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<20), 1<<20)
	failures := 0
	for n := 1; sc.Scan(); n++ {
		line := sc.Text()
		if i := strings.IndexByte(line, '#'); i >= 0 {
			line = line[:i]
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		f := strings.Split(line, ";")
		if len(f) != 5 {
			t.Fatalf("%s:%d: want 5 fields, got %d", path, n, len(f))
		}
		var runes []rune
		for _, h := range strings.Fields(f[0]) {
			v, err := strconv.ParseUint(h, 16, 32)
			if err != nil {
				t.Fatalf("%s:%d: bad code point %q", path, n, h)
			}
			runes = append(runes, rune(v))
		}
		dir := map[string]Direction{"0": LeftToRight, "1": RightToLeft, "2": Auto}[strings.TrimSpace(f[1])]
		wantPara, _ := strconv.Atoi(strings.TrimSpace(f[2]))
		res := Resolve(runes, dir)
		lines++
		var bad string
		if int(res.ParagraphLevel) != wantPara {
			bad = "paragraph level"
		}
		wantLevels := strings.Fields(f[3])
		if len(wantLevels) != len(runes) {
			t.Fatalf("%s:%d: %d levels for %d code points", path, n, len(wantLevels), len(runes))
		}
		for i, w := range wantLevels {
			if w == "x" {
				continue
			}
			if v, _ := strconv.Atoi(w); int(res.Levels[i]) != v && bad == "" {
				bad = "level of character " + strconv.Itoa(i)
			}
		}
		var gotOrder []string
		for _, i := range res.Order {
			if !Removed(ClassOf(runes[i])) {
				gotOrder = append(gotOrder, strconv.Itoa(i))
			}
		}
		if got, want := strings.Join(gotOrder, " "), strings.Join(strings.Fields(f[4]), " "); got != want && bad == "" {
			bad = "visual order: got " + got + ", want " + want
		}
		if bad != "" {
			if failures++; failures <= 10 {
				t.Errorf("%s:%d: %s\n  %s", path, n, bad, strings.TrimSpace(line))
			}
		}
	}
	if failures > 10 {
		t.Errorf("... and %d more failures", failures-10)
	}
	return lines
}

func TestConformanceVendoredSubset(t *testing.T) {
	if n := runConformance(t, "testdata/BidiCharacterTest.subset.txt"); n < 500 {
		t.Errorf("only %d conformance lines ran", n)
	}
}

// BIDICHARACTERTEST names a full BidiCharacterTest.txt to run in place of the
// vendored subset (about 91,000 cases).
func TestConformanceFull(t *testing.T) {
	path := os.Getenv("BIDICHARACTERTEST")
	if path == "" {
		t.Skip("set BIDICHARACTERTEST to a BidiCharacterTest.txt to run the full suite")
	}
	n := runConformance(t, path)
	t.Logf("%d conformance cases", n)
	if n < 90000 {
		t.Errorf("only %d cases ran; is this the full BidiCharacterTest.txt?", n)
	}
}
