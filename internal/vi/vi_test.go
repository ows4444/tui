package vi

import (
	"strings"
	"testing"
)

// buf is a Buffer over a rune slice. Undo restores the text from before the
// group of edits between Begin and the next Begin or End.
type buf struct {
	rs       []rune
	cur      int
	sel      int
	selOn    bool
	undo     [][]rune
	redo     [][]rune
	open     bool
	typedIns bool
}

func (b *buf) Len() int              { return len(b.rs) }
func (b *buf) At(i int) rune         { return b.rs[i] }
func (b *buf) Cursor() int           { return b.cur }
func (b *buf) SetCursor(i int)       { b.cur = i }
func (b *buf) Select(a int, on bool) { b.sel, b.selOn = a, on }
func (b *buf) Begin()                { b.open = false; b.typedIns = false }
func (b *buf) End()                  { b.open = false }
func (b *buf) Replace(from, to int, ins []rune) {
	if !b.open {
		b.undo = append(b.undo, append([]rune(nil), b.rs...))
		b.redo = nil
		b.open = true
	}
	out := append([]rune(nil), b.rs[:from]...)
	out = append(out, ins...)
	b.rs = append(out, b.rs[to:]...)
}
func (b *buf) Undo() {
	if n := len(b.undo); n > 0 {
		b.redo = append(b.redo, b.rs)
		b.rs, b.undo = b.undo[n-1], b.undo[:n-1]
	}
}
func (b *buf) Redo() {
	if n := len(b.redo); n > 0 {
		b.undo = append(b.undo, b.rs)
		b.rs, b.redo = b.redo[n-1], b.redo[:n-1]
	}
}

// run types keys into a buffer holding text, with the cursor at the | in text
// (or at 0), as a widget would: Insert-mode keys the engine does not take are
// typed here. It returns the text with | at the cursor.
func run(t *testing.T, text, keys string) (string, *State, *buf) {
	t.Helper()
	b := &buf{}
	if i := strings.IndexRune(text, '|'); i >= 0 {
		text = text[:i] + text[i+1:]
		b.cur = i
	}
	b.rs = []rune(text)
	s := &State{}
	for _, r := range keys {
		k := Key{R: r}
		switch r {
		case 0x1b:
			k = Key{S: Esc}
		case '\r':
			k = Key{S: Enter}
		case 0x12:
			k = Key{S: Redo}
		case 0x7f:
			k = Key{S: Backspace}
		}
		if res := s.Handle(b, k); !res.Handled {
			s.typeKey(b, k)
		}
	}
	return string(b.rs[:b.cur]) + "|" + string(b.rs[b.cur:]), s, b
}

func TestEditing(t *testing.T) {
	const esc = "\x1b"
	for _, c := range []struct{ name, in, keys, want string }{
		{"x", "|abc", "x", "|bc"},
		{"3x", "|abcdef", "3x", "|def"},
		{"x at end moves left", "ab|c", "x", "a|b"},
		{"X", "ab|c", "X", "a|c"},
		{"dw", "|one two three", "dw", "|two three"},
		{"2dw", "|one two three", "2dw", "|three"},
		{"d2w", "|one two three", "d2w", "|three"},
		{"dw on the last word keeps the line", "one |two\nnext", "dw", "one| \nnext"},
		{"de", "|one two", "de", "| two"},
		{"db", "one |two", "db", "|two"},
		{"d$", "one |two\nx", "d$", "one| \nx"},
		{"D", "one |two\nx", "D", "one| \nx"},
		{"d0", "one |two", "d0", "|two"},
		{"dd", "a\n|b\nc", "dd", "a\n|c"},
		{"dd last line", "a\n|b", "dd", "|a"},
		{"2dd", "|a\nb\nc", "2dd", "|c"},
		{"dj", "|a\nb\nc", "dj", "|c"},
		{"dk", "a\n|b\nc", "dk", "|c"},
		{"dG", "a\n|b\nc", "dG", "|a"},
		{"dgg", "a\n|b\nc", "dgg", "|c"},
		{"cw", "|one two", "cwX" + esc, "|X two"},
		{"cw on a word end acts like ce", "o|ne two", "cwX" + esc, "o|X two"},
		{"cc", "a\n|bcd\ne", "ccX" + esc, "a\n|X\ne"},
		{"C", "ab|cd", "CX" + esc, "ab|X"},
		{"s", "|abc", "sX" + esc, "|Xbc"},
		{"S", "a\n|bcd", "SX" + esc, "a\n|X"},
		{"yw p", "|one two", "ywwP", "one one| two"},
		{"yy p", "|a\nb", "yyp", "a\n|a\nb"},
		{"yy P", "a\n|b", "yyP", "a\n|b\nb"},
		{"yy p on the last line", "|a", "yyp", "a\n|a"},
		{"dd p", "|a\nb", "ddp", "b\n|a"},
		{"x p", "|ab", "xp", "b|a"},
		{"3p", "|a", "yl3p", "aaa|a"},
		{"diw", "one t|wo three", "diw", "one | three"},
		{"daw", "one t|wo three", "daw", "one |three"},
		{"daw at the end takes the space before", "one t|wo", "daw", "on|e"},
		{"ciw", "one t|wo", "ciwX" + esc, "one |X"},
		{"di quote", `say "hel|lo" now`, `di"`, `say "|" now`},
		{"da quote", `say "hel|lo" now`, `da"`, `say | now`},
		{"di paren", "f(a, |b)", "di(", "f(|)"},
		{"da paren nested", "f(g(|x), y)", "da(", "f(g|, y)"},
		{"di brace", "{ |a }", "di{", "{|}"},
		{"i", "a|b", "iX" + esc, "a|Xb"},
		{"a", "a|b", "aX" + esc, "ab|X"},
		{"I", "  a|b", "IX" + esc, "  |Xab"},
		{"A", "a|b", "AX" + esc, "ab|X"},
		{"o", "a|b\nc", "oX" + esc, "ab\n|X\nc"},
		{"O", "a\nb|c", "OX" + esc, "a\n|X\nbc"},
		{"J", "a|b\n  c", "J", "ab| c"},
		{"3J", "a|\nb\nc", "3J", "a b| c"},
		{"r", "|abc", "rx", "|xbc"},
		{"3r", "|abcd", "3rx", "xx|xd"},
		{"r too long", "|ab", "3rx", "|ab"},
		{"~", "|aB", "2~", "A|b"},
		{"insert backspace", "|", "iab\x7f" + esc, "|a"},
		{"insert enter", "|", "ia\rb" + esc, "a\n|b"},
		{"u after cw", "|one two", "cwX" + esc + "u", "|one two"},
		{"u after x", "|abc", "xxu", "|bc"},
		{"u after o", "|a", "ox" + esc + "u", "|a"},
		{"ctrl r", "|abc", "xu\x12", "|bc"},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, _, _ := run(t, c.in, c.keys)
			if got != c.want {
				t.Errorf("%q then %q = %q; want %q", c.in, c.keys, got, c.want)
			}
		})
	}
}

func TestMotions(t *testing.T) {
	for _, c := range []struct{ name, in, keys, want string }{
		{"l", "|abc", "l", "a|bc"},
		{"l stops at the last character", "a|bc", "5l", "ab|c"},
		{"h", "ab|c", "h", "a|bc"},
		{"h stops at the line start", "a\nb|", "5h", "a\n|b"},
		{"0", "ab|c", "0", "|abc"},
		{"^", "  a|b", "^", "  |ab"},
		{"$", "|abc\nd", "$", "ab|c\nd"},
		{"w", "|one two", "w", "one |two"},
		{"w over punctuation", "|a.b", "w", "a|.b"},
		{"W", "|a.b c", "W", "a.b |c"},
		{"w to the next line", "|a\nb", "w", "a\n|b"},
		{"w onto an empty line", "|a\n\nb", "w", "a\n|\nb"},
		{"b", "one |two", "b", "|one two"},
		{"b to the previous line", "a\n|b", "b", "|a\nb"},
		{"e", "|one two", "e", "on|e two"},
		{"2e", "|one two", "2e", "one tw|o"},
		{"j keeps the column", "abcd\nx\nabcd", "lllj", "abcd\n|x\nabcd"},
		{"j j restores the column", "a|bcd\nx\nabcd", "jj", "abcd\nx\na|bcd"},
		{"k", "ab\ncd|", "k", "a|b\ncd"},
		{"j on the last line stays", "a|b", "j", "a|b"},
		{"G", "|a\nb\nc", "G", "a\nb\n|c"},
		{"3G", "|a\nb\nc\nd", "3G", "a\nb\n|c\nd"},
		{"gg", "a\nb\n|c", "gg", "|a\nb\nc"},
		{"f", "|a.b.c", "f.", "a|.b.c"},
		{"2f", "|a.b.c", "2f.", "a.b|.c"},
		{"F", "a.b.|c", "F.", "a.b|.c"},
		{"t", "|a.b", "t.", "|a.b"},
		{"t further", "|ab.c", "t.", "a|b.c"},
		{";", "|a.b.c", "f.;", "a.b|.c"},
		{",", "|a.b.c", "f.;,", "a|.b.c"},
		{"t ;", "|ab.cd.e", "t.;", "ab.c|d.e"},
		{"df", "|a.b.c", "df.", "|b.c"},
		{"dt", "|a.b.c", "dt.", "|.b.c"},
		{"dF", "a.b.|c", "dF.", "a.b|c"},
		{"arrows", "|abc\ndef", "\x00", "|abc\ndef"},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, _, _ := run(t, c.in, c.keys)
			if got != c.want {
				t.Errorf("%q then %q = %q; want %q", c.in, c.keys, got, c.want)
			}
		})
	}
}

func TestDotRepeats(t *testing.T) {
	const esc = "\x1b"
	for _, c := range []struct{ name, in, keys, want string }{
		{"x", "|abcd", "x.", "|cd"},
		{"dw", "|a b c d", "dw.", "|c d"},
		{"cw", "|aa bb cc", "cwX" + esc + "w.", "X |X cc"},
		{"dd", "|a\nb\nc", "dd.", "|c"},
		{"A", "|a\nb", "AX" + esc + "j.", "aX\nb|X"},
		{"o", "|a", "ox" + esc + ".", "a\nx\n|x"},
		{"p", "|a", "ylp.", "aa|a"},
		{"count", "|abcdef", "2x.", "|ef"},
		{"undo then dot", "|abcd", "x.u", "|bcd"},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, _, _ := run(t, c.in, c.keys)
			if got != c.want {
				t.Errorf("%q then %q = %q; want %q", c.in, c.keys, got, c.want)
			}
		})
	}
}

func TestVisual(t *testing.T) {
	const esc = "\x1b"
	for _, c := range []struct{ name, in, keys, want string }{
		{"d", "|abcd", "vld", "|cd"},
		{"x", "|abcd", "vlx", "|cd"},
		{"backwards", "ab|cd", "vhd", "a|d"},
		{"y p", "|ab", "vlyP", "a|bab"},
		{"c", "|abcd", "vlcX" + esc, "|Xcd"},
		{"o", "|abcd", "vlohd", "|cd"},
		{"V d", "a\n|b\nc", "Vd", "a\n|c"},
		{"V j d", "|a\nb\nc", "Vjd", "|c"},
		{"V y p", "|a\nb", "VyP", "|a\na\nb"},
		{"iw", "|one two", "viwd", "| two"},
		{"aw", "|one two", "vawd", "|two"},
		{"~", "|abc", "vl~", "|ABc"},
		{"v again leaves", "|abc", "vv", "|abc"},
		{"esc leaves", "|abc", "vl" + esc, "a|bc"},
		{"D", "a\n|b\nc", "vD", "a\n|c"},
		{"w extends", "|one two", "vwd", "|wo"},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, _, _ := run(t, c.in, c.keys)
			if got != c.want {
				t.Errorf("%q then %q = %q; want %q", c.in, c.keys, got, c.want)
			}
		})
	}
}

func TestModesAndSelection(t *testing.T) {
	_, s, b := run(t, "|abc", "v")
	if s.Mode() != Visual || !b.selOn || b.sel != 0 {
		t.Fatalf("mode %v, selection %v at %d", s.Mode(), b.selOn, b.sel)
	}
	_, s, b = run(t, "|abc", "v\x1b")
	if s.Mode() != Normal || b.selOn {
		t.Fatalf("Esc left mode %v with selection %v", s.Mode(), b.selOn)
	}
	_, s, _ = run(t, "|abc", "i")
	if s.Mode() != Insert {
		t.Fatalf("i left mode %v", s.Mode())
	}
	_, s, _ = run(t, "|abc", "2d")
	if s.Pending() != "2d" {
		t.Fatalf("Pending = %q", s.Pending())
	}
	_, s, _ = run(t, "|abc", "2d\x1b")
	if s.Pending() != "" {
		t.Fatalf("Esc left %q pending", s.Pending())
	}
}

func TestYankReachesTheCaller(t *testing.T) {
	b := &buf{rs: []rune("one two")}
	s := &State{}
	var got Result
	for _, r := range "yw" {
		got = s.Handle(b, Key{R: r})
	}
	if !got.Yanked || got.Yank != "one " {
		t.Fatalf("yank = %+v", got)
	}
	if got = s.Handle(b, Key{R: 'x'}); got.Yanked {
		t.Fatal("x reported a yank")
	}
}

func TestInsertKeysAreLeftToTheWidget(t *testing.T) {
	b := &buf{rs: []rune("ab")}
	s := &State{}
	s.Handle(b, Key{R: 'i'})
	for _, k := range []Key{{R: 'z'}, {S: Left}, {S: Enter}} {
		if s.Handle(b, k).Handled {
			t.Errorf("%+v was taken in Insert mode", k)
		}
	}
	if !s.Handle(b, Key{S: Esc}).Handled || s.Mode() != Normal {
		t.Error("Esc did not leave Insert mode")
	}
}

func TestUnknownKeysAreIgnored(t *testing.T) {
	got, _, _ := run(t, "|abc", "Zq@#")
	if got != "|abc" {
		t.Fatalf("text changed: %q", got)
	}
}

func TestEmptyBuffer(t *testing.T) {
	for _, keys := range []string{"x", "dd", "dw", "yyp", "J", "w", "b", "e", "$", "G", "u", ".", "vd", "Vd", "rx", "~", "ciwX\x1b", "diw", "di(", "ofoo\x1b", "p", "P"} {
		got, _, _ := run(t, "|", keys)
		_ = got // reaching here without a panic is the check
	}
}
