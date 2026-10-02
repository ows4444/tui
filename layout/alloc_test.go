package layout

import (
	"strings"
	"testing"
)

func TestNextLineMatchesSplit(t *testing.T) {
	for _, s := range []string{"", "a", "a\nb", "a\n", "\n", "\n\nx", "a\n\nb\n"} {
		var got []string
		for rest, more := s, true; more; {
			var l string
			l, rest, more = nextLine(rest)
			got = append(got, l)
		}
		if want := strings.Split(s, "\n"); strings.Join(got, "|") != strings.Join(want, "|") || len(got) != len(want) {
			t.Errorf("nextLine(%q) = %q, want %q", s, got, want)
		}
	}
}

func TestBlockRenderClipsAndPads(t *testing.T) {
	got := Block("ab\ncdef\n").Render(Size{W: 3, H: 4})
	if want := "ab \ncde\n   \n   "; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got := Block("").Render(Size{W: 2, H: 2}); got != "  \n  " {
		t.Errorf("empty block: %q", got)
	}
	if s := Block("a\n\nbcd").Measure(Unconstrained()); s != (Size{W: 3, H: 3}) {
		t.Errorf("measure = %v", s)
	}
}

func TestBoxRenderEdgeCases(t *testing.T) {
	got := NewBox().Border(NormalBorder()).Padding(1, 1, 1, 1).Render("a\n\nbc")
	lines := strings.Split(got, "\n")
	if len(lines) != 5+2 {
		t.Fatalf("rows = %d: %q", len(lines), got)
	}
	if NewBox().Render("") != "" {
		t.Errorf("empty unbordered box should be empty")
	}
}

func TestJoinAllocsBounded(t *testing.T) {
	blocks := []string{"ab\ncd\nef", "x\ny", "hello"}
	h := testing.AllocsPerRun(50, func() { _ = joinHorizontalAlign(1, AlignCenter, blocks...) })
	v := testing.AllocsPerRun(50, func() { _ = joinVerticalAlign(1, AlignEnd, blocks...) })
	b := testing.AllocsPerRun(50, func() { _ = Block("ab\ncd\nef").Render(Size{W: 4, H: 4}) })
	if h > 6 || v > 2 || b > 2 {
		t.Errorf("allocs: horizontal %v, vertical %v, block %v", h, v, b)
	}
}
