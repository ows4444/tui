package ansi

import (
	"strings"
	"testing"
)

var (
	bold = NewStyle().Bold()
	red  = NewStyle().Foreground(Red)
)

func lines(ts []Text) []string {
	out := make([]string, len(ts))
	for i, t := range ts {
		out[i] = t.Plain()
	}
	return out
}

func eq(a, b []string) bool {
	return strings.Join(a, "\x00") == strings.Join(b, "\x00") && len(a) == len(b)
}

func TestTextBuildsAndMeasures(t *testing.T) {
	x := NewText("hello ", Style{}).Append("world", bold).Append("!", bold)
	if x.Plain() != "hello world!" || x.Width() != 12 || x.Height() != 1 {
		t.Errorf("Plain %q Width %d Height %d", x.Plain(), x.Width(), x.Height())
	}
	if len(x) != 2 || x[1].Text != "world!" {
		t.Errorf("neighbours of one style were not joined: %+v", x)
	}
	if got, want := x.String(), "hello "+bold.Render("world!"); got != want {
		t.Errorf("String = %q, want %q", got, want)
	}
	if len(NewText("", bold)) != 0 || (Text(nil)).Width() != 0 || (Text(nil)).Height() != 1 {
		t.Error("empty Text")
	}
	multi := NewText("ab\nwide世界\nc", Style{})
	if multi.Width() != 8 || multi.Height() != 3 {
		t.Errorf("multi-line Width %d Height %d", multi.Width(), multi.Height())
	}
}

func TestTextIsSanitisedAndImmutable(t *testing.T) {
	x := NewText("a\x1b[31mb\x1b]52;c;ZXZpbA==\x07c\td", Style{})
	if strings.ContainsRune(x.Plain(), 0x1b) || strings.Contains(x.String(), "\x1b[31m") {
		t.Errorf("an escape survived: %q", x.String())
	}
	if p := x.Plain(); !strings.HasPrefix(p, "abc") || !strings.HasSuffix(p, "d") || strings.ContainsRune(p, '\t') {
		t.Errorf("Plain = %q: escapes should be gone and the tab a run of spaces", p)
	}
	base := NewText("ab", Style{})
	_ = base.Append("cd", bold)
	if base.Plain() != "ab" {
		t.Error("Append changed its receiver")
	}
}

func TestSliceAndTruncateKeepStyles(t *testing.T) {
	x := NewText("abc", Style{}).Append("DEF", bold).Append("ghi", red)
	got := x.Slice(2, 7)
	if got.Plain() != "cDEFg" || len(got) != 3 || got[1].Style != bold {
		t.Errorf("Slice = %+v", got)
	}
	if x.Truncate(4).Plain() != "abcD" || x.Truncate(0).Plain() != "" {
		t.Errorf("Truncate = %q", x.Truncate(4).Plain())
	}
	w := NewText("a世b", Style{})
	if w.Truncate(2).Plain() != "a" || w.Truncate(3).Plain() != "a世" || w.Slice(2, 4).Plain() != "b" {
		t.Errorf("a wide character straddling an edge: %q %q %q", w.Truncate(2).Plain(), w.Truncate(3).Plain(), w.Slice(2, 4).Plain())
	}
	if multi := NewText("abcd\nefgh", Style{}).Truncate(2); multi.Plain() != "ab\nef" {
		t.Errorf("Truncate on each line = %q", multi.Plain())
	}
}

func TestWrapBreaksAtSpaces(t *testing.T) {
	x := NewText("hello world foo", Style{})
	for _, c := range []struct {
		w    int
		want []string
	}{
		{11, []string{"hello world", "foo"}},
		{5, []string{"hello", "world", "foo"}},
		{100, []string{"hello world foo"}},
		{0, []string{"hello world foo"}},
	} {
		if got := lines(x.Wrap(c.w)); !eq(got, c.want) {
			t.Errorf("Wrap(%d) = %q, want %q", c.w, got, c.want)
		}
	}
}

func TestWrapCutsLongWordsKeepsIndentAndParagraphs(t *testing.T) {
	if got := lines(NewText("abcdefghij", Style{}).Wrap(4)); !eq(got, []string{"abcd", "efgh", "ij"}) {
		t.Errorf("long word = %q", got)
	}
	if got := lines(NewText("  ab cd", Style{}).Wrap(4)); !eq(got, []string{"  ab", "cd"}) {
		t.Errorf("indent = %q", got)
	}
	if got := lines(NewText("a   b", Style{}).Wrap(3)); !eq(got, []string{"a", "b"}) {
		t.Errorf("space run at a break = %q", got)
	}
	if got := lines(NewText("one two\n\nthree", Style{}).Wrap(5)); !eq(got, []string{"one", "two", "", "three"}) {
		t.Errorf("paragraphs = %q", got)
	}
	if got := lines(NewText("ab ", Style{}).Wrap(5)); !eq(got, []string{"ab"}) {
		t.Errorf("trailing space = %q", got)
	}
	if got := lines(NewText("a世b世", Style{}).Wrap(3)); !eq(got, []string{"a世", "b世"}) {
		t.Errorf("wide characters = %q", got)
	}
	if got := Text(nil).Wrap(5); len(got) != 1 || got[0].Plain() != "" {
		t.Errorf("empty = %+v", got)
	}
}

func TestWrapKeepsEachPiecesStyleAcrossABreak(t *testing.T) {
	x := NewText("plain ", Style{}).Append("bold words here", bold)
	got := x.Wrap(10)
	want := []string{"plain bold", "words here"}
	if !eq(lines(got), want) {
		t.Fatalf("lines = %q", lines(got))
	}
	if got[0][1].Style != bold || got[0][1].Text != "bold" || got[1][0].Style != bold {
		t.Errorf("styles lost across the break: %+v", got)
	}
	for i, l := range got {
		if Width(l.String()) > 10 || Width(l.Plain()) != l.Width() {
			t.Errorf("line %d: width %d", i, l.Width())
		}
	}
}

func TestWrapProperty(t *testing.T) {
	words := []string{"a", "bb", "ccc", "dddd", "世", "世界", "e\u0301", "👨\u200d👩", "x", "  ", " "}
	var seed uint32 = 12345
	next := func(n int) int { seed = seed*1664525 + 1013904223; return int(seed>>16) % n }
	for iter := 0; iter < 300; iter++ {
		var sb strings.Builder
		for k := next(12); k >= 0; k-- {
			sb.WriteString(words[next(len(words))])
			if next(4) == 0 {
				sb.WriteByte(' ')
			}
			if next(15) == 0 {
				sb.WriteByte('\n')
			}
		}
		text := NewText(sb.String(), bold)
		for _, w := range []int{1, 2, 3, 5, 9} {
			var joined []string
			for _, l := range text.Wrap(w) {
				if l.Width() > w && !(w == 1 && l.Width() == 2) && !(w < 2 && l.Width() <= 2) {
					t.Fatalf("width %d: line %q is %d wide (text %q)", w, l.Plain(), l.Width(), sb.String())
				}
				joined = append(joined, strings.ReplaceAll(l.Plain(), " ", ""))
			}
			if got, want := strings.Join(joined, ""), strings.ReplaceAll(strings.ReplaceAll(text.Plain(), " ", ""), "\n", ""); got != want {
				t.Fatalf("width %d lost or reordered text:\n got %q\nwant %q", w, got, want)
			}
		}
	}
}
