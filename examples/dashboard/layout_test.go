package main

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

var update = flag.Bool("update", false, "rewrite the golden files")

// The screen is built with layout.Node and pads no text by hand.
func TestScreenUsesLayoutNodeAndNoManualPadding(t *testing.T) {
	b, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(b)
	if !strings.Contains(src, "layout.Draw(") {
		t.Error("main.go does not build its view with layout.Draw")
	}
	if !strings.Contains(src, "layout.GridNodeGaps(") {
		t.Error("main.go does not build its tables with layout.GridNodeGaps")
	}
	for _, banned := range []string{"JoinHorizontal", "JoinVertical", "%-5s", "%-13s", "statusLine", "metricLine", "Row of Columns"} {
		if strings.Contains(src, banned) {
			t.Errorf("main.go still pads or joins by hand: %s", banned)
		}
	}
}

// At any terminal size the screen fills exactly the space it is given, with
// every row the same width, so a parent can place it anywhere.
func TestScreenRendersExactlyTheRequestedSize(t *testing.T) {
	m := initialModel()
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

// The natural size of the screen is what View draws: before the terminal
// width is known that is the narrow, one-column layout.
func TestViewIsTheScreensNaturalSize(t *testing.T) {
	m := initialModel()
	want := m.screen().Measure(layout.Loose(layout.Size{W: mediumMin - 1, H: layout.Unbounded}))
	lines := strings.Split(m.View(), "\n")
	if len(lines) != want.H {
		t.Errorf("View is %d rows, Measure says %d", len(lines), want.H)
	}
	for i, l := range lines {
		if w := ansi.Width(l); w != want.W {
			t.Errorf("row %d is %d wide, Measure says %d", i, w, want.W)
		}
	}
}

// The tables line up: every row of a table puts its columns at the same
// offsets, which the old hand-padded version did not (the Disk value sat two
// columns left of CPU and Mem).
func TestTableColumnsAlign(t *testing.T) {
	plain := strings.Split(ansi.StripANSI(initialModel().View()), "\n")
	// display column (not byte offset: the graphs use multi-byte glyphs) at
	// which marker starts on the first row containing prefix.
	col := func(prefix, marker string) int {
		for _, l := range plain {
			if strings.Contains(l, prefix) {
				i := strings.Index(l, marker)
				if i < 0 {
					return -1
				}
				return ansi.Width(l[:i])
			}
		}
		t.Fatalf("no row containing %q", prefix)
		return -1
	}
	if a, b, c := col("healthy", "healthy"), col("degraded", "degraded"), col("down", "down"); a != b || b != c || a < 0 {
		t.Errorf("badges start at columns %d, %d and %d, want the same", a, b, c)
	}
	if cpu, mem, disk := col("CPU", "%"), col("Mem", "%"), col("Disk", "%"); cpu != mem || mem != disk || cpu < 0 {
		t.Errorf("value columns end at %d, %d and %d, want the same", cpu, mem, disk)
	}
}

// Golden view, ANSI stripped so it stays readable. Regenerate with -update.
func TestViewGolden(t *testing.T) {
	got := ansi.StripANSI(initialModel().View())
	golden := filepath.Join("testdata", "view.golden")
	if *update {
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

// plainAt renders the screen at exactly w columns wide and the height it
// measures for w, checking every row is w wide.
func plainAt(t *testing.T, w int) string {
	t.Helper()
	s := initialModel().screen()
	size := s.Measure(layout.Loose(layout.Size{W: w, H: layout.Unbounded}))
	size.W = w
	lines := strings.Split(s.Render(size), "\n")
	if len(lines) != size.H {
		t.Errorf("width %d: %d rows, want %d", w, len(lines), size.H)
	}
	for i, l := range lines {
		if got := ansi.Width(l); got != w {
			t.Errorf("width %d: row %d is %d wide, want %d", w, i, got, w)
		}
	}
	return ansi.StripANSI(strings.Join(lines, "\n"))
}

// At 40 columns the dashboard is one column; at 120 it is the grid.
func TestResponsiveGoldens(t *testing.T) {
	for _, tc := range []struct {
		w    int
		file string
		wide bool
	}{{40, "view40.golden", false}, {120, "view120.golden", true}} {
		got := plainAt(t, tc.w)
		golden := filepath.Join("testdata", tc.file)
		if *update {
			if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		want, err := os.ReadFile(golden)
		if err != nil {
			t.Fatal(err)
		}
		if got != string(want) {
			t.Errorf("view at %d differs from %s (go test -update to regenerate):\n%s", tc.w, golden, got)
		}
		// stacked: metrics divider sits below the status rows; grid: the
		// status and metrics tables share rows.
		sameRow := false
		for _, l := range strings.Split(got, "\n") {
			if strings.Contains(l, "Database") && strings.Contains(l, "CPU") {
				sameRow = true
			}
		}
		if sameRow != tc.wide {
			t.Errorf("width %d: status and metrics share a row = %v, want %v", tc.w, sameRow, tc.wide)
		}
	}
}

// renderAt renders the screen at exactly s, ANSI stripped.
func renderAt(t *testing.T, s layout.Size) string {
	t.Helper()
	return ansi.StripANSI(initialModel().screen().Render(s))
}

var sizeGoldens = []struct {
	size layout.Size
	file string
}{
	{layout.Size{W: 40, H: 10}, "view40x10.golden"},
	{layout.Size{W: 80, H: 24}, "view80x24.golden"},
	{layout.Size{W: 300, H: 80}, "view300x80.golden"},
}

// Goldens at three terminal sizes; regenerate with -update.
func TestSizeGoldens(t *testing.T) {
	for _, tc := range sizeGoldens {
		got := renderAt(t, tc.size)
		golden := filepath.Join("testdata", tc.file)
		if *update {
			if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		want, err := os.ReadFile(golden)
		if err != nil {
			t.Fatal(err)
		}
		if got != string(want) {
			t.Errorf("%v differs from %s (go test -update to regenerate):\n%s", tc.size, golden, got)
		}
	}
}

// The three sizes pick three different layouts.
func TestLayoutDiffersAcrossSizes(t *testing.T) {
	small := renderAt(t, layout.Size{W: 40, H: 10})
	mid := renderAt(t, layout.Size{W: 80, H: 24})
	big := renderAt(t, layout.Size{W: 300, H: 80})
	if small == mid || mid == big || small == big {
		t.Error("40x10, 80x24 and 300x80 must render different layouts")
	}
	rowHas := func(s string, a, b string) bool {
		for _, l := range strings.Split(s, "\n") {
			if strings.Contains(l, a) && strings.Contains(l, b) {
				return true
			}
		}
		return false
	}
	if rowHas(small, "API", "Deploying") || !rowHas(mid, "API", "Deploy") {
		t.Error("status and deploy should share a row at 80 columns but not at 40")
	}
}
