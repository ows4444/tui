package drawer

import (
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui/ansi"
)

const (
	baseW = 60
	baseH = 20
)

func dotBase() string {
	row := strings.Repeat(".", baseW)
	rows := make([]string, baseH)
	for i := range rows {
		rows[i] = row
	}
	return strings.Join(rows, "\n")
}

var edges = []struct {
	name string
	edge Edge
}{{"right", EdgeRight}, {"left", EdgeLeft}, {"top", EdgeTop}, {"bottom", EdgeBottom}}

// at returns an open drawer of the given edge with hidden set as a slide would
// have left it.
func at(edge Edge, hidden float64) Model {
	m := New("Settings\nTheme: dark\nFont: 14")
	m.Edge, m.Width, m.Height = edge, 24, 6
	m.hidden = hidden
	return m
}

// covered counts the cells of the rendered base that differ from the plain
// dotted base, i.e. how much of the drawer is visible.
func covered(out string) int {
	n := 0
	for _, l := range strings.Split(out, "\n") {
		for _, r := range ansi.StripANSI(l) {
			if r != '.' {
				n++
			}
		}
	}
	return n
}

// At every progress value, for every edge, the rendered view has the base's
// rows and the base's width: the slide never grows or shrinks the screen.
func TestSlideRenderKeepsTheBaseSize(t *testing.T) {
	base := dotBase()
	for _, e := range edges {
		for i := 0; i <= 40; i++ {
			h := float64(i) / 40
			lines := strings.Split(at(e.edge, h).Render(base), "\n")
			if len(lines) != baseH {
				t.Fatalf("%s hidden=%.3f: %d rows, want %d", e.name, h, len(lines), baseH)
			}
			for r, l := range lines {
				if w := ansi.Width(l); w != baseW {
					t.Fatalf("%s hidden=%.3f: row %d is %d wide, want %d", e.name, h, r, w, baseW)
				}
			}
		}
	}
}

// Fully in place (hidden 0) is the drawer as it has always been drawn; fully
// out (hidden 1) shows nothing but the base.
func TestSlideRenderEndpoints(t *testing.T) {
	base := dotBase()
	for _, e := range edges {
		m := at(e.edge, 0)
		plain := m
		plain.SlideDuration = 0
		if m.Render(base) != plain.Render(base) {
			t.Errorf("%s: hidden=0 differs from the at-rest drawer", e.name)
		}
		if covered(m.Render(base)) == 0 {
			t.Errorf("%s: the drawer at rest draws nothing", e.name)
		}
		if got := at(e.edge, 1).Render(base); got != base {
			t.Errorf("%s: hidden=1 shows part of the drawer:\n%s", e.name, got)
		}
	}
}

// As the slide progresses the visible part of the drawer never shrinks.
func TestSlideRenderVisiblePartGrowsMonotonically(t *testing.T) {
	base := dotBase()
	for _, e := range edges {
		prev := -1
		for i := 40; i >= 0; i-- { // hidden 1 -> 0
			n := covered(at(e.edge, float64(i)/40).Render(base))
			if n < prev {
				t.Fatalf("%s: visible cells fell from %d to %d at hidden=%.3f", e.name, prev, n, float64(i)/40)
			}
			prev = n
		}
		if prev == 0 {
			t.Errorf("%s: nothing was ever visible", e.name)
		}
	}
}

// The drawer emerges from its own edge: the visible part touches that edge of
// the screen while it is partly out.
func TestSlideRenderEmergesFromItsEdge(t *testing.T) {
	base := dotBase()
	col := func(lines []string, x int) string {
		var b strings.Builder
		for _, l := range lines {
			b.WriteString(string([]rune(ansi.StripANSI(l))[x]))
		}
		return b.String()
	}
	for _, e := range edges {
		lines := strings.Split(at(e.edge, 0.5).Render(base), "\n")
		touches := false
		switch e.edge {
		case EdgeRight:
			touches = strings.Trim(col(lines, baseW-1), ".") != ""
		case EdgeLeft:
			touches = strings.Trim(col(lines, 0), ".") != ""
		case EdgeTop:
			touches = strings.Trim(ansi.StripANSI(lines[0]), ".") != ""
		case EdgeBottom:
			touches = strings.Trim(ansi.StripANSI(lines[baseH-1]), ".") != ""
		}
		if !touches {
			t.Errorf("%s: at hidden=0.5 the visible part does not touch its own edge:\n%s", e.name, strings.Join(lines, "\n"))
		}
		// And it is not yet all the way in: less is covered than at rest.
		if covered(strings.Join(lines, "\n")) >= covered(at(e.edge, 0).Render(base)) {
			t.Errorf("%s: at hidden=0.5 as much is visible as at rest", e.name)
		}
	}
}

// A drawer wider or taller than the base is clipped to it while it slides, so
// the view never grows.
func TestSlideRenderClipsADrawerLargerThanTheBase(t *testing.T) {
	base := dotBase()
	for _, e := range edges {
		m := at(e.edge, 0.3)
		m.Width, m.Height = 200, 60
		lines := strings.Split(m.Render(base), "\n")
		if len(lines) != baseH {
			t.Fatalf("%s: %d rows", e.name, len(lines))
		}
		for r, l := range lines {
			if ansi.Width(l) != baseW {
				t.Fatalf("%s: row %d is %d wide", e.name, r, ansi.Width(l))
			}
		}
	}
}

// Driven by real slide ticks, the rendered drawer grows from nothing to the
// at-rest drawer.
func TestSlideInRendersFromNothingToFullyOpen(t *testing.T) {
	base := dotBase()
	for _, e := range edges {
		m := New("Settings\nTheme: dark")
		m.Edge, m.Width, m.Height = e.edge, 24, 6
		m.Hide()
		m.SlideDuration = d
		rest := m
		rest.Show()
		rest.SlideDuration = 0
		m.SlideIn()
		m, _ = tickAt(m, t0)
		if got := m.Render(base); got != base {
			t.Errorf("%s: the first frame of a slide in already shows the drawer", e.name)
		}
		prev := 0
		for i := 1; i <= 10; i++ {
			m, _ = tickAt(m, t0.Add(d*time.Duration(i)/10))
			n := covered(m.Render(base))
			if n < prev {
				t.Fatalf("%s: visible cells fell from %d to %d during a slide in", e.name, prev, n)
			}
			prev = n
		}
		if got, want := m.Render(base), rest.Render(base); got != want {
			t.Errorf("%s: after the slide in the drawer differs from the at-rest one", e.name)
		}
	}
}
