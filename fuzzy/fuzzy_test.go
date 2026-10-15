package fuzzy

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"unicode"
)

func TestScoreMatches(t *testing.T) {
	cases := []struct {
		q, text string
		pos     []int
	}{
		{"cp", "CommandPalette", []int{0, 7}},
		{"CP", "commandpalette", []int{0, 7}},
		{"go", "main.go", []int{5, 6}},
		{"fb", "foo_bar", []int{0, 4}},
		{"héllo", "Héllo wörld", []int{0, 1, 2, 3, 4}},
		{"ÉA", "éa", []int{0, 1}},
	}
	for _, c := range cases {
		_, pos, ok := Score(c.q, c.text)
		if !ok || !reflect.DeepEqual(pos, c.pos) {
			t.Errorf("Score(%q, %q) = %v, %v; want %v", c.q, c.text, pos, ok, c.pos)
		}
	}
}

func TestScoreRejects(t *testing.T) {
	for _, c := range [][2]string{{"xyz", "abc"}, {"ab", "ba"}, {"abc", "ab"}, {"a", ""}} {
		if _, _, ok := Score(c[0], c[1]); ok {
			t.Errorf("Score(%q, %q) matched", c[0], c[1])
		}
	}
}

func TestEmptyQueryMatchesAll(t *testing.T) {
	sc, pos, ok := Score("", "anything")
	if !ok || sc != 0 || pos != nil {
		t.Errorf("empty query = %d, %v, %v", sc, pos, ok)
	}
	got := Find("", 3, func(i int) string { return fmt.Sprint("t", 3-i) })
	if len(got) != 3 || got[0].Index != 0 || got[2].Index != 2 {
		t.Errorf("empty query Find = %+v", got)
	}
}

func TestRankingOrder(t *testing.T) {
	// A prefix beats a word start, which beats a scattered match.
	texts := []string{"sxxexxt", "reset", "settings", "set"}
	got := Find("set", len(texts), func(i int) string { return texts[i] })
	var order []string
	for _, r := range got {
		order = append(order, texts[r.Index])
	}
	want := []string{"set", "settings", "reset", "sxxexxt"}
	if !reflect.DeepEqual(order, want) {
		t.Errorf("order = %v, want %v", order, want)
	}
}

func TestConsecutiveBeatsScattered(t *testing.T) {
	a, _, _ := Score("abc", "abcxxx")
	b, _, _ := Score("abc", "axbxcx")
	if a <= b {
		t.Errorf("consecutive %d should beat scattered %d", a, b)
	}
}

func TestWordBoundaryBeatsMidWord(t *testing.T) {
	a, _, _ := Score("b", "foo bar")
	b, _, _ := Score("b", "foobar")
	if a <= b {
		t.Errorf("word start %d should beat mid-word %d", a, b)
	}
}

func TestFindStableAndShorterFirst(t *testing.T) {
	texts := []string{"abcd", "abc", "abc"}
	got := Find("abc", 3, func(i int) string { return texts[i] })
	if got[0].Index != 1 || got[1].Index != 2 || got[2].Index != 0 {
		t.Errorf("order = %v %v %v", got[0].Index, got[1].Index, got[2].Index)
	}
}

func TestHighlight(t *testing.T) {
	mark := func(s string) string { return "[" + s + "]" }
	if got := Highlight("CommandPalette", []int{0, 7}, mark); got != "[C]ommand[P]alette" {
		t.Errorf("got %q", got)
	}
	if got := Highlight("abcd", []int{1, 2}, mark); got != "a[bc]d" {
		t.Errorf("adjacent runes should share a mark: %q", got)
	}
	if got := Highlight("abc", []int{-1, 9}, mark); got != "abc" {
		t.Errorf("out of range: %q", got)
	}
	if got := Highlight("héllo", []int{1}, mark); got != "h[é]llo" {
		t.Errorf("multibyte: %q", got)
	}
}

func TestLongTextStillMatches(t *testing.T) {
	text := strings.Repeat("x", 2000) + "needle"
	_, pos, ok := Score("needle", text)
	if !ok || len(pos) != 6 || pos[0] != 2000 {
		t.Errorf("long text = %v, %v", pos, ok)
	}
}

// FuzzScore checks the contract for any input: a match is exactly a
// subsequence, and positions are ascending, in range and spell the query.
func FuzzScore(f *testing.F) {
	for _, s := range [][2]string{{"cp", "CommandPalette"}, {"", "x"}, {"é", "É"}, {"ab", "ba"}} {
		f.Add(s[0], s[1])
	}
	f.Fuzz(func(t *testing.T, q, text string) {
		_, pos, ok := Score(q, text)
		qr, tr := []rune(q), []rune(text)
		if ok != isSubseq(qr, tr) {
			t.Fatalf("ok=%v but subsequence=%v for %q in %q", ok, isSubseq(qr, tr), q, text)
		}
		if !ok || len(qr) == 0 {
			return
		}
		if len(pos) != len(qr) {
			t.Fatalf("%d positions for %d query runes", len(pos), len(qr))
		}
		for i, p := range pos {
			if p < 0 || p >= len(tr) || (i > 0 && p <= pos[i-1]) {
				t.Fatalf("bad positions %v for %q in %q", pos, q, text)
			}
			if unicode.ToLower(tr[p]) != unicode.ToLower(qr[i]) {
				t.Fatalf("position %d is %q, query rune is %q", p, tr[p], qr[i])
			}
		}
	})
}

func isSubseq(q, t []rune) bool {
	i := 0
	for _, c := range t {
		if i < len(q) && unicode.ToLower(c) == unicode.ToLower(q[i]) {
			i++
		}
	}
	return i == len(q)
}

func BenchmarkFind(b *testing.B) {
	texts := make([]string, 1000)
	for i := range texts {
		texts[i] = fmt.Sprintf("internal/module%03d/handler_%d.go", i%97, i)
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		Find("mod hnd go", len(texts), func(i int) string { return texts[i] })
	}
}

func Example() {
	names := []string{"Open File", "Close Window", "Command Palette", "Copy Path"}
	for _, r := range Find("cp", len(names), func(i int) string { return names[i] }) {
		fmt.Println(Highlight(names[r.Index], r.Positions, strings.ToUpper))
	}
	// Output:
	// Copy Path
	// Command Palette
}
