package main

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

var updateGolden = flag.Bool("update", false, "rewrite the golden files")

// The click regions come from the layout, not from numbers worked out by hand.
func TestNoHandComputedTabRegions(t *testing.T) {
	b, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(b)
	if strings.Contains(src, "tabRegions") {
		t.Error("main.go still has the hand-computed tabRegions")
	}
	if !strings.Contains(src, "layout.RectOf(") {
		t.Error("main.go does not read the tab rectangles with layout.RectOf")
	}
	if !strings.Contains(src, "layout.Draw(") {
		t.Error("main.go does not build its view with layout.Draw")
	}
}

// Each rectangle the layout reports is exactly where its tab is drawn: the
// cells under it, read back from the rendered view, are " label ".
func TestTabRectsMatchTheRenderedBar(t *testing.T) {
	for _, active := range []int{0, 1} {
		m := initialModel()
		m.tabs.SetActive(active)
		root := m.screen()
		size := root.Measure(layout.Unconstrained())
		lines := strings.Split(ansi.StripANSI(layout.Draw(root, layout.Unconstrained())), "\n")
		for i, label := range tabLabels {
			r, ok := layout.RectOf(root, size, tabName(i))
			if !ok {
				t.Fatalf("no rectangle for tab %d", i)
			}
			if r.H != 1 || r.W != len(label)+2 {
				t.Errorf("tab %d rect = %+v, want a 1 x %d cell", i, r, len(label)+2)
			}
			row := []rune(lines[r.Y])
			want := " " + label + " "
			if i == active {
				want = "[" + label + "]"
			}
			if got := string(row[r.X : r.X+r.W]); got != want {
				t.Errorf("active=%d: cells under tab %d's rect are %q, want %q", active, i, got, want)
			}
		}
	}
}

// Every cell of every tab hits that tab, and nothing else on the bar does.
func TestEveryCellOfATabHitsIt(t *testing.T) {
	m := initialModel()
	root := m.screen()
	size := root.Measure(layout.Unconstrained())
	hits := 0
	for x := 0; x < size.W; x++ {
		for y := 0; y < size.H; y++ {
			ev := tui.MouseEvent{X: x, Y: y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress}
			got, ok := m.tabAt(ev)
			if !ok {
				continue
			}
			hits++
			r, _ := layout.RectOf(root, size, tabName(got))
			if x < r.X || x >= r.X+r.W || y < r.Y || y >= r.Y+r.H {
				t.Errorf("(%d,%d) reported tab %d whose rect is %+v", x, y, got, r)
			}
		}
	}
	want := 0
	for _, l := range tabLabels {
		want += len(l) + 2
	}
	if hits != want {
		t.Errorf("%d cells hit a tab, want %d (one row of each tab)", hits, want)
	}
}

// The bar drawn from per-tab nodes is byte-identical to tabs.Model's own bar,
// so repeating its styling here cannot drift unnoticed.
func TestTabBarMatchesTabsModelView(t *testing.T) {
	for _, active := range []int{0, 1} {
		m := initialModel()
		m.tabs.SetActive(active)
		got := layout.Draw(m.tabBar(), layout.Unconstrained())
		if want := m.tabs.View(); got != want {
			t.Errorf("active=%d:\n got %q\nwant %q", active, got, want)
		}
	}
}

// At any terminal size the screen fills exactly the space it is given.
func TestScreenRendersExactlyTheRequestedSize(t *testing.T) {
	for _, active := range []int{0, 1} {
		m := initialModel()
		m.tabs.SetActive(active)
		for _, s := range []layout.Size{{W: 40, H: 10}, {W: 80, H: 24}, {W: 300, H: 80}, {W: 5, H: 3}} {
			lines := strings.Split(m.screen().Render(s), "\n")
			if len(lines) != s.H {
				t.Fatalf("%v: %d rows, want %d", s, len(lines), s.H)
			}
			for i, l := range lines {
				if w := ansi.Width(l); w != s.W {
					t.Errorf("%v: row %d is %d wide, want %d", s, i, w, s.W)
				}
			}
		}
	}
}

// Every row of the natural view is the same width: the key hints no longer
// spill past the right border.
func TestNaturalViewHasNoOverflowingRow(t *testing.T) {
	lines := strings.Split(initialModel().View(), "\n")
	want := ansi.Width(lines[0])
	for i, l := range lines {
		if w := ansi.Width(l); w != want {
			t.Errorf("row %d is %d wide, the top border is %d: %q", i, w, want, ansi.StripANSI(l))
		}
	}
}

// Golden view for each tab, ANSI stripped so it stays readable. Regenerate
// with -update.
func TestViewGolden(t *testing.T) {
	for active, name := range []string{"general", "advanced"} {
		m := initialModel()
		m.tabs.SetActive(active)
		got := ansi.StripANSI(m.View())
		golden := filepath.Join("testdata", name+".golden")
		if *updateGolden {
			if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		want, err := os.ReadFile(golden)
		if err != nil {
			t.Fatal(err)
		}
		if got != string(want) {
			t.Errorf("view differs from %s (go test -update to regenerate):\n%s", golden, got)
		}
	}
}
