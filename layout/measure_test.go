package layout

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
)

// recorder logs Measure/Render order so tests can assert the two-pass
// contract.
type recorder struct {
	log  *[]string
	name string
	want Size
}

func (r recorder) Measure(c Constraints) Size {
	*r.log = append(*r.log, "measure:"+r.name)
	return c.Constrain(r.want)
}
func (r recorder) Render(s Size) string {
	*r.log = append(*r.log, "render:"+r.name)
	return Block("").Render(s)
}

func TestConstrainClampsBothAxes(t *testing.T) {
	c := Constraints{MinW: 5, MaxW: 10, MinH: 2, MaxH: 3}
	cases := []struct{ in, want Size }{
		{Size{1, 1}, Size{5, 2}},
		{Size{7, 2}, Size{7, 2}},
		{Size{99, 99}, Size{10, 3}},
	}
	for _, tc := range cases {
		if got := c.Constrain(tc.in); got != tc.want {
			t.Errorf("Constrain(%v) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestConstrainToleratesDegenerateConstraints(t *testing.T) {
	for _, c := range []Constraints{
		{}, {MinW: -5, MaxW: -1, MinH: -2, MaxH: -9}, {MinW: 9, MaxW: 3}, Unconstrained(), Loose(Size{}),
	} {
		got := c.Constrain(Size{4, 4})
		if got.W < 0 || got.H < 0 {
			t.Errorf("%+v -> negative size %v", c, got)
		}
	}
}

func TestDrawMeasuresBeforeRenderingAndFitsConstraints(t *testing.T) {
	var log []string
	root := recorder{log: &log, name: "root", want: Size{20, 5}}
	out := Draw(root, Loose(Size{8, 3}))
	if len(log) != 2 || log[0] != "measure:root" || log[1] != "render:root" {
		t.Fatalf("call order = %v, want measure then render", log)
	}
	lines := strings.Split(out, "\n")
	if len(lines) != 3 || ansi.Width(lines[0]) != 8 {
		t.Errorf("Draw output is %dx%d, want 8x3 (constrained)", ansi.Width(lines[0]), len(lines))
	}
}

func TestBlockMeasureReportsNaturalSize(t *testing.T) {
	b := Block("ab\n\x1b[31mcdef\x1b[0m\n你")
	if got := b.Measure(Unconstrained()); got != (Size{4, 3}) {
		t.Errorf("Measure = %v, want {4 3}", got)
	}
	if got := b.Measure(Loose(Size{2, 2})); got != (Size{2, 2}) {
		t.Errorf("constrained Measure = %v, want {2 2}", got)
	}
	if got := Block("").Measure(Unconstrained()); got != (Size{}) {
		t.Errorf("empty Measure = %v, want zero", got)
	}
}

func TestBlockRenderPadsAndClipsToExactSize(t *testing.T) {
	out := Block("ab\ncdef").Render(Size{3, 3})
	want := "ab \ncde\n   "
	if out != want {
		t.Errorf("Render = %q, want %q", out, want)
	}
	// A wide rune straddling the cut is dropped, then padded back to width.
	if got := Block("a你").Render(Size{2, 1}); ansi.Width(got) != 2 {
		t.Errorf("wide-rune clip width = %d, want 2 (%q)", ansi.Width(got), got)
	}
	for _, s := range []Size{{0, 3}, {3, 0}, {-1, -1}} {
		if got := Block("x").Render(s); got != "" {
			t.Errorf("Render(%v) = %q, want empty", s, got)
		}
	}
}

func checkSolve(t *testing.T, avail int, items []flexSpec) []int {
	t.Helper()
	got := solveFlex(avail, items)
	if len(got) != len(items) {
		t.Fatalf("len = %d, want %d", len(got), len(items))
	}
	for i, it := range items {
		lo := clampAxis(it.min, 0, Unbounded)
		hi := clampAxis(it.max, lo, Unbounded)
		if got[i] < lo || got[i] > hi {
			t.Fatalf("item %d size %d outside [%d,%d] (avail %d, items %+v)", i, got[i], lo, hi, avail, items)
		}
	}
	return got
}

func TestSolveFlexGrowsBySlackWeight(t *testing.T) {
	got := checkSolve(t, 100, []flexSpec{
		{basis: 10, max: Unbounded},
		{basis: 0, grow: 1, max: Unbounded},
		{basis: 0, grow: 3, max: Unbounded},
	})
	// 90 spare split 1:3 -> 22.5/67.5 with cumulative rounding, sum exact.
	if got[0] != 10 || got[1]+got[2] != 90 || got[2] < 3*got[1]-3 {
		t.Errorf("sizes = %v", got)
	}
}

func TestSolveFlexGrowRespectsMaxAndRedistributes(t *testing.T) {
	got := checkSolve(t, 100, []flexSpec{
		{grow: 1, max: 10},
		{grow: 1, max: Unbounded},
	})
	if got[0] != 10 || got[1] != 90 {
		t.Errorf("sizes = %v, want [10 90]", got)
	}
}

func TestSolveFlexShrinksByWeightedBasisDownToMin(t *testing.T) {
	got := checkSolve(t, 60, []flexSpec{
		{basis: 40, shrink: 1, min: 35, max: Unbounded},
		{basis: 40, shrink: 1, min: 0, max: Unbounded},
	})
	// 20 overflow: even split would be 10 each; first floors at 35 (-5),
	// remaining 15 lands on the second.
	if got[0] != 35 || got[1] != 25 {
		t.Errorf("sizes = %v, want [35 25]", got)
	}
	// shrink=0 items never give up space.
	got = checkSolve(t, 10, []flexSpec{{basis: 8, max: Unbounded}, {basis: 8, shrink: 1, max: Unbounded}})
	if got[0] != 8 || got[1] != 2 {
		t.Errorf("sizes = %v, want [8 2]", got)
	}
}

func TestSolveFlexEmptyAndDegenerate(t *testing.T) {
	if got := solveFlex(10, nil); len(got) != 0 {
		t.Errorf("empty items -> %v", got)
	}
	got := checkSolve(t, -5, []flexSpec{{basis: 3, grow: 1, shrink: 1, min: -2, max: -1}})
	if got[0] != 0 {
		t.Errorf("negative avail/bounds -> %v, want [0]", got)
	}
}

// FuzzSolveFlex checks the solver terminates and keeps every size within
// its bounds; when there is room it also fills avail exactly.
func FuzzSolveFlex(f *testing.F) {
	f.Add(100, []byte{10, 1, 0, 0, 255, 5, 0, 1, 2, 40})
	f.Fuzz(func(t *testing.T, avail int, raw []byte) {
		avail %= 100000
		var items []flexSpec
		for i := 0; i+4 < len(raw) && len(items) < 12; i += 5 {
			max := int(raw[i+4])
			if max == 255 {
				max = Unbounded
			}
			items = append(items, flexSpec{
				basis: int(raw[i]), grow: int(raw[i+1] % 8), shrink: int(raw[i+2] % 8),
				min: int(raw[i+3] % 50), max: max,
			})
		}
		got := checkSolve(t, avail, items)

		// If avail is reachable purely by growing (or shrinking) and some
		// item can still absorb, the total must match exactly.
		sum := 0
		for _, g := range got {
			sum += g
		}
		want := max(avail, 0)
		if sum < want { // undershoot only allowed when nobody can still grow
			for i, it := range items {
				hi := clampAxis(it.max, clampAxis(it.min, 0, Unbounded), Unbounded)
				if it.grow > 0 && got[i] < hi {
					t.Fatalf("undershoot %d < %d though item %d can grow: %+v -> %v", sum, want, i, items, got)
				}
			}
		}
		if sum > want { // overshoot only allowed when nobody can still shrink
			for i, it := range items {
				lo := clampAxis(it.min, 0, Unbounded)
				if it.shrink > 0 && got[i] > lo {
					t.Fatalf("overshoot %d > %d though item %d can shrink: %+v -> %v", sum, want, i, items, got)
				}
			}
		}
	})
}

// FuzzBlockRender checks Render always yields exactly W x H display cells.
func FuzzBlockRender(f *testing.F) {
	f.Add("ab\n\x1b[31mcd\x1b[0m\n你好", 3, 4)
	f.Fuzz(func(t *testing.T, s string, w, h int) {
		w, h = w%60, h%20
		out := Block(s).Render(Size{w, h})
		if w <= 0 || h <= 0 {
			if out != "" {
				t.Fatalf("Render(%d,%d) = %q, want empty", w, h, out)
			}
			return
		}
		lines := strings.Split(out, "\n")
		if len(lines) != h {
			t.Fatalf("%d rows, want %d", len(lines), h)
		}
		for i, l := range lines {
			if ansi.Width(l) != w {
				t.Fatalf("row %d width %d, want %d (%q)", i, ansi.Width(l), w, l)
			}
		}
	})
}
