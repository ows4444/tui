package layout

import (
	"strings"
	"testing"
)

func TestDrawTightFillAbsorbsRemainingSpace(t *testing.T) {
	root := Row(0, FlexChild{Node: Block("ab")}, Fill(Block("x")))
	if got := Draw(root, Loose(Size{30, 2})); len(strings.Split(got, "\n")[0]) >= 30 {
		t.Fatalf("precondition: Draw unchanged, got %q", got)
	}
	out := DrawTight(root, Size{30, 4})
	assertExact(t, out, Size{30, 4})
	if Tight(Size{3, 4}) != (Constraints{MinW: 3, MaxW: 3, MinH: 4, MaxH: 4}) {
		t.Error("Tight bounds")
	}
	col := DrawTight(Column(0, FlexChild{Node: Block("h")}, Fill(Block("b")), FlexChild{Node: Block("f")}), Size{5, 10})
	assertExact(t, col, Size{5, 10})
	rows := strings.Split(col, "\n")
	if strings.TrimSpace(rows[0]) != "h" || strings.TrimSpace(rows[9]) != "f" {
		t.Errorf("column = %q", col)
	}
}

func TestPctAndFrAllocateProportionally(t *testing.T) {
	r := Row(0,
		FlexChild{Node: Block("a"), BasisLen: Pct(25)},
		FlexChild{Node: Block("b"), BasisLen: Pct(75)},
	)
	rs := r.(flexNode).plan(Size{40, 1}).sizes
	if rs[0] != 10 || rs[1] != 30 {
		t.Errorf("pct sizes = %v", rs)
	}
	f := Row(0,
		FlexChild{Node: Block("fixed"), Basis: 10},
		FlexChild{Node: Block("a"), BasisLen: Fr(1)},
		FlexChild{Node: Block("b"), BasisLen: Fr(3)},
	)
	fs := f.(flexNode).plan(Size{50, 1}).sizes
	if fs[0] != 10 || fs[1] != 10 || fs[2] != 30 {
		t.Errorf("fr sizes = %v", fs)
	}
	assertExact(t, f.Render(Size{50, 2}), Size{50, 2})
	// Percent is of space after gaps.
	g := Row(2, FlexChild{Node: Block("a"), BasisLen: Pct(50)}, FlexChild{Node: Block("b"), BasisLen: Pct(50)})
	if s := g.(flexNode).plan(Size{22, 1}).sizes; s[0] != 10 || s[1] != 10 {
		t.Errorf("gap pct = %v", s)
	}
}

func TestScrollWindowsChild(t *testing.T) {
	child := Block("1\n2\n3\n4\n5")
	if got := Scroll(child, 0).Render(Size{2, 2}); got != "1 \n2 " {
		t.Errorf("offset 0 = %q", got)
	}
	if got := Scroll(child, 2).Render(Size{2, 2}); got != "3 \n4 " {
		t.Errorf("offset 2 = %q", got)
	}
	if got := Scroll(child, 99).Render(Size{2, 2}); got != "4 \n5 " {
		t.Errorf("clamped = %q", got)
	}
	if got := Scroll(child, -3).Render(Size{2, 2}); got != "1 \n2 " {
		t.Errorf("negative = %q", got)
	}
	if got := Scroll(child, 0).Measure(Loose(Size{9, 3})); got != (Size{1, 3}) {
		t.Errorf("Measure = %v", got)
	}
	if got := Scroll(child, 0).Render(Size{0, 2}); got != "" {
		t.Errorf("zero = %q", got)
	}
	// Inside a Fill it takes the leftover height.
	out := DrawTight(Column(0, Fill(Scroll(child, 1))), Size{3, 3})
	assertExact(t, out, Size{3, 3})
}
