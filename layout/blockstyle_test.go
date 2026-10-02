package layout

import (
	"strings"
	"testing"
)

func TestBlockLinesAreSelfContainedAcrossOpenStyle(t *testing.T) {
	b := Block("\x1b[31mone\ntwo\nthree\x1b[0m")
	lines := strings.Split(b.Render(Size{6, 3}), "\n")
	want := []string{
		"\x1b[31mone\x1b[0m   ",
		"\x1b[31mtwo\x1b[0m   ",
		"\x1b[31mthree\x1b[0m ",
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i, lines[i], want[i])
		}
	}
}

func TestBlockOpenStyleDoesNotBleedIntoSiblingInRow(t *testing.T) {
	// The first Block never closes its style; the sibling must stay plain.
	r := Row(1,
		FlexChild{Node: Block("\x1b[31mred\nstill red")},
		FlexChild{Node: Block("aa\nbb")},
	)
	out := r.Render(Size{12, 2})
	for i, l := range strings.Split(out, "\n") {
		idx := strings.LastIndex(l, "\x1b[0m")
		if idx < 0 {
			t.Fatalf("row %d never resets the open style: %q", i, l)
		}
		if strings.Contains(l[idx:], "\x1b[31m") {
			t.Errorf("row %d: style reopened after reset: %q", i, l)
		}
		if !strings.Contains(l[idx:], []string{"aa", "bb"}[i]) {
			t.Errorf("row %d: sibling cells not after reset: %q", i, l)
		}
	}
}

func TestBlockCarriesLinkAndPlainBlockUnchanged(t *testing.T) {
	b := Block("\x1b]8;;http://x\x1b\\ab\ncd\x1b]8;;\x1b\\")
	lines := strings.Split(b.Render(Size{3, 2}), "\n")
	if want := "\x1b]8;;http://x\x1b\\ab\x1b]8;;\x1b\\ "; lines[0] != want {
		t.Errorf("line 0 = %q, want %q", lines[0], want)
	}
	if want := "\x1b]8;;http://x\x1b\\cd\x1b]8;;\x1b\\ "; lines[1] != want {
		t.Errorf("line 1 = %q, want %q", lines[1], want)
	}
	if got := Block("ab\ncd").Render(Size{3, 2}); got != "ab \ncd " {
		t.Errorf("plain block = %q", got)
	}
}
