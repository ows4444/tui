package cellbuf_test

import (
	"errors"
	"math/rand"
	"strings"
	"testing"

	"github.com/ows4444/tui/cellbuf"
	"github.com/ows4444/tui/internal/vtscreen"
)

const screenCols = 64

// screenOf draws s on a fresh screen, one row per line, each row ended with a
// reset (the parser's rows are style independent).
func screenOf(s string, rows int) *vtscreen.Screen {
	sc := vtscreen.NewScreen(screenCols, rows)
	for i, ln := range strings.Split(s, "\n") {
		if i > 0 {
			sc.Write([]byte("\r\n"))
		}
		sc.Write([]byte(ln + "\x1b[0m"))
	}
	return sc
}

func sameScreen(t *testing.T, a, b *vtscreen.Screen, rows int, what string) {
	t.Helper()
	for y := 0; y < rows; y++ {
		for x := 0; x < screenCols; x++ {
			if ca, cb := a.Cell(x, y), b.Cell(x, y); ca != cb {
				t.Fatalf("%s: cell (%d,%d): %+v != %+v", what, x, y, ca, cb)
			}
		}
	}
	la, lb := a.Lines(), b.Lines()
	for i := range la {
		if la[i] != lb[i] {
			t.Fatalf("%s: line %d: %q != %q", what, i, la[i], lb[i])
		}
	}
}

func sameCells(t *testing.T, a, b *cellbuf.Buffer, what string) {
	t.Helper()
	if a.Width() != b.Width() || a.Height() != b.Height() {
		t.Fatalf("%s: size %dx%d != %dx%d", what, a.Width(), a.Height(), b.Width(), b.Height())
	}
	for y := 0; y < a.Height(); y++ {
		for x := 0; x < a.Width(); x++ {
			ca, cb := a.At(x, y), b.At(x, y)
			if ca.Cluster != cb.Cluster || ca.Width != cb.Width || a.Style(ca.Style) != b.Style(cb.Style) {
				t.Fatalf("%s: cell (%d,%d): %+v/%+v != %+v/%+v", what, x, y, ca, a.Style(ca.Style), cb, b.Style(cb.Style))
			}
		}
	}
}

func roundTrip(t *testing.T, s string) {
	t.Helper()
	b, err := cellbuf.Parse(s)
	if err != nil {
		t.Fatalf("Parse(%q): %v", s, err)
	}
	out := b.String()
	rows := b.Height()
	sameScreen(t, screenOf(s, rows), screenOf(out, rows), rows, "screen of "+out)
	b2, err := cellbuf.Parse(out)
	if err != nil {
		t.Fatalf("re-Parse(%q): %v", out, err)
	}
	sameCells(t, b, b2, "cells of "+out)
	if out2 := b2.String(); out2 != out {
		t.Fatalf("String not stable: %q then %q", out, out2)
	}
}

var corpus = []string{
	"", "plain", "  padded  ", "a\nb\nc",
	"\x1b[1mbold\x1b[0m plain", "\x1b[31;1mred bold\x1b[22m red\x1b[39m",
	"\x1b[38;5;196mx\x1b[48;5;21my\x1b[0m", "\x1b[38;2;1;2;3mrgb\x1b[48;2;255;0;128mbg",
	"\x1b[4:3mcurly\x1b[24m", "\x1b[90;107mbright",
	"\x1b[3;4;5;7;8;9mall attrs", "\x1b[1mopen style\nnext row",
	"日本語 mixed", "a日b", "é combining", "👨‍👩‍👧 zwj", "🇯🇵 flag", "👍🏽 skin",
	"\x1b[31m日本\x1b[0m語", "x\x1b]8;;https://example.com\x1b\\link\x1b]8;;\x1b\\y",
	"\x1b]8;id=1;http://a.b\x07text\x1b]8;;\x07", "\x1b[1m\x1b]8;;http://a\x1b\\bold link\x1b]8;;\x1b\\ tail",
	"line\r\nwith crlf", "\x1b[2mfaint\x1b[0m\x1b[1;3;4mstack",
}

func TestRoundTripCorpus(t *testing.T) {
	for _, s := range corpus {
		roundTrip(t, s)
	}
}

func TestRoundTripRandom(t *testing.T) {
	frags := []string{
		"a", "Z", " ", "  ", "日", "本", "é", "é", "👍", "👨‍👩‍👧", "🇯🇵", "~",
		"\x1b[0m", "\x1b[1m", "\x1b[22m", "\x1b[3m", "\x1b[4m", "\x1b[4:3m", "\x1b[24m", "\x1b[7m", "\x1b[27m",
		"\x1b[31m", "\x1b[41m", "\x1b[91m", "\x1b[102m", "\x1b[39m", "\x1b[49m", "\x1b[38;5;99m", "\x1b[48;2;9;8;7m",
		"\x1b]8;;http://x.y/z\x1b\\", "\x1b]8;;\x1b\\", "\n",
	}
	rng := rand.New(rand.NewSource(24))
	accepted := 0
	for i := 0; i < 3000; i++ {
		var sb strings.Builder
		for n := rng.Intn(14); n >= 0; n-- {
			sb.WriteString(frags[rng.Intn(len(frags))])
		}
		s := sb.String()
		if _, err := cellbuf.Parse(s); err != nil {
			if !errors.Is(err, cellbuf.ErrUnsupported) {
				t.Fatalf("Parse(%q): %v", s, err)
			}
			continue
		}
		accepted++
		roundTrip(t, s)
	}
	if accepted < 2000 {
		t.Fatalf("only %d of 3000 random strings were accepted; the generator is off", accepted)
	}
}

func TestParseRejects(t *testing.T) {
	for _, s := range []string{"\x1b[2Jx", "\x1b[99mx", "a\x01b", "\x1b]0;title\x07"} {
		if _, err := cellbuf.Parse(s); !errors.Is(err, cellbuf.ErrUnsupported) {
			t.Errorf("Parse(%q) error = %v, want ErrUnsupported", s, err)
		}
	}
}

// checkWhole fails when a row holds half of a wide cluster.
func checkWhole(t *testing.T, b *cellbuf.Buffer, what string) {
	t.Helper()
	for y := 0; y < b.Height(); y++ {
		for x := 0; x < b.Width(); x++ {
			c := b.At(x, y)
			switch {
			case c.Width == 2:
				if x+1 >= b.Width() {
					t.Fatalf("%s: wide head at the last column (%d,%d)", what, x, y)
				}
				if n := b.At(x+1, y); n.Width != 0 || n.Cluster != "" {
					t.Fatalf("%s: wide head at (%d,%d) has no continuation: %+v", what, x, y, n)
				}
			case c.Width == 0:
				if x == 0 || b.At(x-1, y).Width != 2 {
					t.Fatalf("%s: orphan continuation at (%d,%d)", what, x, y)
				}
			case c.Width != 1 || c.Cluster == "":
				t.Fatalf("%s: bad cell at (%d,%d): %+v", what, x, y, c)
			}
		}
	}
}

func TestSetStringWideAtLastColumn(t *testing.T) {
	for w := 1; w <= 6; w++ {
		for x := 0; x <= w; x++ {
			b := cellbuf.New(w, 1)
			n := b.SetString(x, 0, "日日日", 0)
			checkWhole(t, b, "plain")
			if n > w {
				t.Fatalf("w=%d x=%d wrote %d columns", w, x, n)
			}
			// The cluster that does not fit is not split: the column is blank.
			if x == w-1 {
				if c := b.At(w-1, 0); c.Cluster != " " || c.Width != 1 {
					t.Fatalf("last column = %+v, want a blank", c)
				}
			}
		}
	}
	b := cellbuf.New(3, 1)
	id := b.StyleID(cellbuf.Style{FG: cellbuf.Basic(1)})
	b.SetString(0, 0, "a日日", id)
	checkWhole(t, b, "styled")
	if got := b.At(0, 0).Cluster + b.At(1, 0).Cluster + b.At(2, 0).Cluster; got != "a日" {
		t.Fatalf("row = %q, want a日 (3rd column is blank, so a日 plus...)", got)
	}
}

func TestSetStringWideAtLastColumnInSub(t *testing.T) {
	b := cellbuf.New(8, 2)
	v := b.Sub(cellbuf.Rect{X: 2, Y: 1, W: 3, H: 1})
	v.SetString(0, 0, "ab日", 0)
	checkWhole(t, b, "sub")
	if b.At(5, 1).Cluster != " " || b.At(4, 1).Cluster != " " || b.At(3, 1).Cluster != "b" {
		t.Fatalf("sub write leaked or split: %q", b.Lines())
	}
	// Clipped left: a wide cluster cut by the left edge shows as a blank.
	v.SetString(-1, 0, "日x", 0)
	checkWhole(t, b, "sub left")
	if v.At(0, 0).Cluster != " " || v.At(1, 0).Cluster != "x" {
		t.Fatalf("left clip: %q", v.Lines())
	}
}

func TestOverwriteHalfOfWideCluster(t *testing.T) {
	b := cellbuf.New(6, 1)
	b.SetString(0, 0, "日日日", 0)
	b.SetString(1, 0, "x", 0) // lands on the continuation of the first
	checkWhole(t, b, "over continuation")
	if b.At(0, 0).Cluster != " " || b.At(1, 0).Cluster != "x" {
		t.Fatalf("got %q", b.Lines())
	}
	b.SetString(2, 0, "y", 0) // lands on a head
	checkWhole(t, b, "over head")
	if b.At(3, 0).Cluster != " " {
		t.Fatalf("got %q", b.Lines())
	}
}

func TestSetStringRandomNeverSplits(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	alpha := []string{"a", "日", "é", "é", "👨‍👩‍👧", "🇯🇵", " ", "\t", "\x1b[1m"}
	for i := 0; i < 2000; i++ {
		b := cellbuf.New(1+rng.Intn(9), 3)
		for k := 0; k < 4; k++ {
			var sb strings.Builder
			for n := rng.Intn(8); n > 0; n-- {
				sb.WriteString(alpha[rng.Intn(len(alpha))])
			}
			target := b
			if rng.Intn(2) == 0 {
				target = b.Sub(cellbuf.Rect{X: rng.Intn(4), Y: rng.Intn(2), W: 1 + rng.Intn(6), H: 2})
			}
			target.SetString(rng.Intn(7)-2, rng.Intn(3), sb.String(), 0)
			checkWhole(t, b, "random")
		}
	}
}

func TestSetStringFiltersControls(t *testing.T) {
	b := cellbuf.New(10, 1)
	n := b.SetString(0, 0, "a\x1b[31mb\tc\nd\x07", 0)
	if n != 4 || strings.TrimRight(b.String(), " ") != "abcd" {
		t.Fatalf("n=%d %q", n, b.String())
	}
}

func TestSetStringKeepsTrailingSpaces(t *testing.T) {
	b := cellbuf.New(6, 1)
	id := b.StyleID(cellbuf.Style{BG: cellbuf.Basic(4)})
	if n := b.SetString(0, 0, "日  ", id); n != 4 {
		t.Fatalf("n = %d", n)
	}
	if c := b.At(3, 0); c.Style != id {
		t.Fatalf("trailing space lost its style: %+v", c)
	}
}

func TestFillAndSub(t *testing.T) {
	b := cellbuf.New(5, 3)
	id := b.StyleID(cellbuf.Style{Attrs: cellbuf.AttrBold, FG: cellbuf.RGB(1, 2, 3)})
	b.Fill(cellbuf.Rect{X: 1, Y: 1, W: 3, H: 5}, "*", id)
	for y := 0; y < 3; y++ {
		for x := 0; x < 5; x++ {
			in := y == 1 || y == 2
			in = in && x >= 1 && x <= 3
			if got := b.At(x, y).Cluster == "*"; got != in {
				t.Fatalf("(%d,%d) filled=%v want %v", x, y, got, in)
			}
		}
	}
	b.Fill(cellbuf.Rect{X: 0, Y: 0, W: 5, H: 1}, "日", id)
	checkWhole(t, b, "wide fill")
	if b.At(4, 0).Cluster != " " || b.At(4, 0).Style != id {
		t.Fatalf("leftover column = %+v", b.At(4, 0))
	}
	sub := b.Sub(cellbuf.Rect{X: 3, Y: 1, W: 9, H: 9})
	if sub.Width() != 2 || sub.Height() != 2 {
		t.Fatalf("sub = %dx%d", sub.Width(), sub.Height())
	}
	if e := b.Sub(cellbuf.Rect{X: 9, Y: 9, W: 2, H: 2}); e.Width() != 0 || e.Height() != 0 {
		t.Fatalf("empty sub = %dx%d", e.Width(), e.Height())
	}
	sub.Clear()
	if b.At(3, 1).Cluster != " " || b.At(2, 1).Cluster != "*" {
		t.Fatalf("Clear through Sub: %q", b.Lines())
	}
	if c := sub.At(5, 5); c.Cluster != " " {
		t.Fatalf("off-grid At = %+v", c)
	}
}

func TestSetStyledAndLinks(t *testing.T) {
	b := cellbuf.New(10, 1)
	n, err := b.SetStyled(1, 0, "\x1b[1mhi\x1b[0m \x1b]8;;http://e.x\x1b\\go\x1b]8;;\x1b\\")
	if err != nil || n != 5 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	if st := b.Style(b.At(1, 0).Style); st.Attrs != cellbuf.AttrBold {
		t.Fatalf("style = %+v", st)
	}
	if st := b.Style(b.At(4, 0).Style); st.Link != "http://e.x" {
		t.Fatalf("link = %+v", st)
	}
	if _, err := b.SetStyled(0, 0, "a\x01b"); !errors.Is(err, cellbuf.ErrUnsupported) {
		t.Fatalf("err = %v", err)
	}
	if _, err := b.SetStyled(0, 0, "a\nb"); !errors.Is(err, cellbuf.ErrUnsupported) {
		t.Fatalf("err = %v", err)
	}
	roundTrip(t, b.String())
}

func TestStyleIDSanitisesLink(t *testing.T) {
	b := cellbuf.New(1, 1)
	id := b.StyleID(cellbuf.Style{Link: "http://a\x1b\\evil\x07"})
	if got := b.Style(id).Link; strings.ContainsAny(got, "\x1b\x07") {
		t.Fatalf("link kept controls: %q", got)
	}
	if b.StyleID(cellbuf.Style{}) != 0 {
		t.Fatal("zero style must be id 0")
	}
}

func TestColonColourParses(t *testing.T) {
	b, err := cellbuf.Parse("\x1b[38:2::10:20:30mx")
	if err != nil {
		t.Fatal(err)
	}
	if got := b.Style(b.At(0, 0).Style).FG; got != cellbuf.RGB(10, 20, 30) {
		t.Fatalf("fg = %#x", uint32(got))
	}
}

func TestTabsExpand(t *testing.T) {
	b, err := cellbuf.Parse("a\tb")
	if err != nil || b.Width() != 9 || b.At(8, 0).Cluster != "b" {
		t.Fatalf("w=%d err=%v", b.Width(), err)
	}
}
