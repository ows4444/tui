package chart

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// #408: bar filled length proportional to Value scaled against max Value.
func TestBarChartProportionalToMax(t *testing.T) {
	items := []BarItem{
		{Label: "a", Value: 50},
		{Label: "b", Value: 100},
	}
	width := 20
	got := BarChart(items, width, theme.DarkTheme())
	lines := strings.Split(got, "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %q", len(lines), got)
	}

	labelWidth := 1 // both labels are width 1
	barWidth := width - labelWidth - 1

	wantLine := func(label string, value, max float64) string {
		filled := int(value/max*float64(barWidth) + 0.5)
		pad := strings.Repeat(" ", labelWidth-ansi.Width(label)+1)
		fill := ansi.NewStyle().Foreground(theme.DarkTheme().Primary).Render(strings.Repeat("█", filled))
		track := ansi.NewStyle().Foreground(theme.DarkTheme().Muted).Render(strings.Repeat("░", barWidth-filled))
		return label + pad + fill + track
	}

	if want := wantLine("a", 50, 100); lines[0] != want {
		t.Errorf("row 0 = %q, want %q", lines[0], want)
	}
	if want := wantLine("b", 100, 100); lines[1] != want {
		t.Errorf("row 1 = %q, want %q", lines[1], want)
	}
}

// #409: max Value <= 0 renders zero-length bars, no divide-by-zero/panic/garbage.
func TestBarChartNonPositiveMax(t *testing.T) {
	tests := []struct {
		name  string
		items []BarItem
	}{
		{"all zero", []BarItem{{Label: "a", Value: 0}, {Label: "b", Value: 0}}},
		{"all negative", []BarItem{{Label: "a", Value: -5}, {Label: "b", Value: -1}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BarChart(tt.items, 20, theme.DarkTheme())
			lines := strings.Split(got, "\n")
			if len(lines) != len(tt.items) {
				t.Fatalf("expected %d lines, got %d: %q", len(tt.items), len(lines), got)
			}
			for i, line := range lines {
				if strings.Contains(line, "█") {
					t.Errorf("row %d = %q, expected zero-length (no filled block)", i, line)
				}
				if ansi.Width(line) > 20 {
					t.Errorf("row %d width %d exceeds 20", i, ansi.Width(line))
				}
			}
		})
	}
}

// #410: labels are left-aligned and padded to a common width so every
// row's bar starts at the same column, matching widgets.KeyValue.
func TestBarChartLabelAlignment(t *testing.T) {
	items := []BarItem{
		{Label: "x", Value: 10},
		{Label: "longlabel", Value: 20},
	}
	got := BarChart(items, 40, theme.DarkTheme())
	lines := strings.Split(got, "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %q", len(lines), got)
	}

	labelWidth := ansi.Width("longlabel")
	for i, it := range items {
		pad := strings.Repeat(" ", labelWidth-ansi.Width(it.Label)+1)
		prefix := it.Label + pad
		if !strings.HasPrefix(lines[i], prefix) {
			t.Errorf("row %d = %q, want prefix %q", i, lines[i], prefix)
		}
	}

}

// #411: every rendered row's ansi.Width stays within the given width.
func TestBarChartWidthBound(t *testing.T) {
	items := []BarItem{
		{Label: "short", Value: 3},
		{Label: "a-much-longer-label", Value: 9},
	}
	for _, width := range []int{5, 10, 15, 25, 40} {
		got := BarChart(items, width, theme.DarkTheme())
		for i, line := range strings.Split(got, "\n") {
			if w := ansi.Width(line); w > width {
				t.Errorf("width=%d row %d: ansi.Width = %d, want <= %d (line %q)", width, i, w, width, line)
			}
		}
	}
}

// #412: empty items slice renders as "" without panicking.
func TestBarChartEmpty(t *testing.T) {
	if got := BarChart(nil, 20, theme.DarkTheme()); got != "" {
		t.Errorf("BarChart(nil, ...) = %q, want empty", got)
	}
	if got := BarChart([]BarItem{}, 20, theme.DarkTheme()); got != "" {
		t.Errorf("BarChart([]BarItem{}, ...) = %q, want empty", got)
	}
}
