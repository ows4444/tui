package layout

import (
	"strings"
	"testing"
)

func chips(words ...string) []Node {
	out := make([]Node, len(words))
	for i, w := range words {
		out[i] = Block("[" + w + "]")
	}
	return out
}

func TestWrapBreaksRowsAtTheWidth(t *testing.T) {
	w := Wrap(1, 0, chips("go", "tui", "cli", "docs")...)
	got := w.Render(Size{W: 14, H: 3})
	want := "[go] [tui]    \n[cli] [docs]  \n              "
	if got != want {
		t.Fatalf("got\n%q\nwant\n%q", got, want)
	}
	if m := w.Measure(Loose(Size{W: 14, H: 10})); m != (Size{W: 12, H: 2}) {
		t.Fatalf("Measure = %v; want 12x2", m)
	}
	if m := w.Measure(Unconstrained()); m != (Size{W: 4 + 1 + 5 + 1 + 5 + 1 + 6, H: 1}) {
		t.Fatalf("one row when unbounded: %v", m)
	}
}

func TestWrapRowGapAndTallChildren(t *testing.T) {
	w := Wrap(2, 1, Block("aa\nbb"), Block("cc"), Block("dd"))
	got := strings.Split(w.Render(Size{W: 8, H: 5}), "\n")
	want := []string{"aa  cc  ", "bb      ", "        ", "dd      ", "        "}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}

func TestWrapOversizeChildAndEmpties(t *testing.T) {
	w := Wrap(1, 0, Block("abcdefghij"), Block(""), Block("x"))
	got := strings.Split(w.Render(Size{W: 5, H: 2}), "\n")
	if got[0] != "abcde" || got[1] != "x    " {
		t.Fatalf("%q", got)
	}
	if Wrap(1, 0).Render(Size{W: 4, H: 2}) != "    \n    " {
		t.Fatal("empty Wrap")
	}
	if Wrap(-3, -3, Block("a"), Block("b")).Render(Size{W: 4, H: 1}) != "ab  " {
		t.Fatal("negative gaps count as 0")
	}
}

func TestWrapRects(t *testing.T) {
	w := Wrap(1, 0, Named("a", Block("aaa")), Named("b", Block("bbb")), Named("c", Block("ccc")))
	for _, tt := range []struct {
		x, y int
		want string
	}{{0, 0, "a"}, {4, 0, "b"}, {1, 1, "c"}} {
		if name, _, _, ok := NamedAt(w, Size{W: 8, H: 2}, tt.x, tt.y); !ok || name != tt.want {
			t.Errorf("(%d,%d) = %q %v; want %q", tt.x, tt.y, name, ok, tt.want)
		}
	}
}
