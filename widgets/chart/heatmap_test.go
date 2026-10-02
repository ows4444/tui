package chart

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// TestHeatMapEmptyGrid proves criterion #427: an empty grid (nil, no rows,
// or every row with zero columns) renders "" without panicking.
func TestHeatMapEmptyGrid(t *testing.T) {
	cases := [][][]float64{
		nil,
		{},
		{{}},
		{{}, {}},
	}
	for _, values := range cases {
		got := HeatMap(values, theme.DarkTheme())
		if got != "" {
			t.Errorf("HeatMap(%v) = %q, want empty", values, got)
		}
	}
}

// TestHeatMapFlatGridUsesMidShade proves criterion #425: a flat grid
// (min == max across every value) renders a uniform mid-level shade for
// every cell, not the max or min level.
func TestHeatMapFlatGridUsesMidShade(t *testing.T) {
	cases := [][][]float64{
		{{5, 5}, {5, 5}},
		{{0}},
		{{1, 1, 1}},
	}
	mid := string(heatShades[len(heatShades)/2])
	for _, values := range cases {
		got := HeatMap(values, theme.DarkTheme())
		for i, row := range values {
			line := ansi.StripANSI(strings.Split(got, "\n")[i])
			want := strings.Repeat(mid, len(row))
			if line != want {
				t.Errorf("HeatMap(%v) row %d = %q, want %q (flat -> uniform mid shade)", values, i, line, want)
			}
		}
	}
}

// TestHeatMapNormalizesAcrossWholeGrid proves criterion #424: each cell is
// normalized between the grid's overall min and max (not a per-row one),
// picking a shading character from the density scale.
func TestHeatMapNormalizesAcrossWholeGrid(t *testing.T) {
	// Overall min 0, max 4 across both rows. A per-row normalization would
	// make row 1's [3,4] look identical to row 0's [0,1] (each spanning
	// its own row's min..max); a whole-grid normalization must not.
	values := [][]float64{
		{0, 1},
		{3, 4},
	}
	got := HeatMap(values, theme.DarkTheme())
	lines := strings.Split(got, "\n")
	if len(lines) != 2 {
		t.Fatalf("HeatMap(%v) has %d lines, want 2", values, len(lines))
	}
	row0 := lines[0]
	row1 := lines[1]
	if row0 == row1 {
		t.Errorf("HeatMap(%v) row0 = %q and row1 = %q are equal, want whole-grid normalization to distinguish them", values, row0, row1)
	}

	// The lowest value (0) should render as the lowest shade level, and the
	// highest value (4) as the highest shade level, given the whole-grid
	// min/max span [0,4].
	if !strings.Contains(row0, string(heatShades[0])) {
		t.Errorf("HeatMap(%v) row0 = %q, want it to contain the lowest shade %q for the grid min", values, row0, string(heatShades[0]))
	}
	if !strings.Contains(row1, string(heatShades[len(heatShades)-1])) {
		t.Errorf("HeatMap(%v) row1 = %q, want it to contain the highest shade %q for the grid max", values, row1, string(heatShades[len(heatShades)-1]))
	}
}

// TestHeatMapRaggedRows proves criterion #426: rows of differing lengths
// each render using their own length, without panicking.
func TestHeatMapRaggedRows(t *testing.T) {
	values := [][]float64{
		{1, 2, 3},
		{1},
		{1, 2},
	}
	got := HeatMap(values, theme.DarkTheme())
	lines := strings.Split(got, "\n")
	if len(lines) != len(values) {
		t.Fatalf("HeatMap(%v) has %d lines, want %d", values, len(lines), len(values))
	}
	for i, row := range values {
		if w := ansi.Width(lines[i]); w != len(row) {
			t.Errorf("HeatMap(%v) row %d width = %d, want %d (row's own length)", values, i, w, len(row))
		}
	}
}

// TestHeatMapRowWidthMatchesCellCount proves criterion #428: each rendered
// row's ansi.Width equals that row's own cell count exactly, one shading
// character per cell with no extra padding.
func TestHeatMapRowWidthMatchesCellCount(t *testing.T) {
	values := [][]float64{
		{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		{-5, 0, 5},
		{42},
	}
	got := HeatMap(values, theme.DarkTheme())
	lines := strings.Split(got, "\n")
	for i, row := range values {
		if w := ansi.Width(lines[i]); w != len(row) {
			t.Errorf("HeatMap row %d ansi.Width = %d, want %d", i, w, len(row))
		}
	}
}
