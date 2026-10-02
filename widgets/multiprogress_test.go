package widgets

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

func TestMultiProgressEmptyItems(t *testing.T) {
	// Criterion #482: empty items renders "" without panicking.
	got := MultiProgress(nil, 40, theme.DarkTheme())
	if got != "" {
		t.Fatalf("MultiProgress(nil, ...) = %q, want empty string", got)
	}

	got = MultiProgress([]ProgressItem{}, 40, theme.DarkTheme())
	if got != "" {
		t.Fatalf("MultiProgress([]ProgressItem{}, ...) = %q, want empty string", got)
	}
}

func TestMultiProgressRowsComposeLabelStatusBar(t *testing.T) {
	// Criterion #479: one row per item, label aligned to a common column
	// (KeyValue convention), a status indicator, and a ProgressBar.
	items := []ProgressItem{
		{Label: "download", Percent: 0.5, Status: ProgressRunning},
		{Label: "build", Percent: 1, Status: ProgressDone},
		{Label: "a-much-longer-label", Percent: 0, Status: ProgressPending},
	}
	tt := theme.DarkTheme()
	got := MultiProgress(items, 60, tt)

	lines := strings.Split(got, "\n")
	if len(lines) != len(items) {
		t.Fatalf("got %d rows, want %d", len(lines), len(items))
	}

	for i, it := range items {
		if !strings.Contains(lines[i], it.Label) {
			t.Errorf("row %d = %q, want it to contain label %q", i, lines[i], it.Label)
		}
		if !strings.Contains(lines[i], it.Status.String()) {
			t.Errorf("row %d = %q, want it to contain status text %q", i, lines[i], it.Status.String())
		}
		// A rendered ProgressBar uses these fill/track glyphs; some bar
		// content should be present once the fixed label/status overhead
		// leaves room, which it does at width 60 for these labels.
		if !strings.ContainsAny(lines[i], "█░") {
			t.Errorf("row %d = %q, want it to contain progress bar glyphs", i, lines[i])
		}
	}

	// Labels are padded to a common column: every row's status indicator
	// (the "●" dot) starts at the same rune offset, following KeyValue's
	// alignment convention.
	dotCol := -1
	for i, l := range lines {
		col := strings.Index(l, items[i].Status.String())
		if col == -1 {
			t.Fatalf("row %d = %q, want its status text", i, l)
		}
		if dotCol == -1 {
			dotCol = col
		} else if col != dotCol {
			t.Errorf("row %d status dot at column %d, want column %d (common with other rows)", i, col, dotCol)
		}
	}
}

func TestMultiProgressStackedNoAggregateRow(t *testing.T) {
	// Criterion #480: rows stack vertically via layout.JoinVertical, with no
	// extra aggregate/combined-total row.
	items := []ProgressItem{
		{Label: "one", Percent: 0.25, Status: ProgressRunning},
		{Label: "two", Percent: 0.75, Status: ProgressRunning},
	}
	got := MultiProgress(items, 50, theme.DarkTheme())

	lines := strings.Split(got, "\n")
	if len(lines) != len(items) {
		t.Fatalf("got %d lines, want exactly %d (one per item, no aggregate row)", len(lines), len(items))
	}
}

func TestMultiProgressRowWidthBounded(t *testing.T) {
	// Criterion #481: every rendered row's ansi.Width stays within width.
	items := []ProgressItem{
		{Label: "short", Percent: 0.1, Status: ProgressPending},
		{Label: "a-somewhat-longer-task-label", Percent: 0.5, Status: ProgressRunning},
		{Label: "x", Percent: 1, Status: ProgressDone},
		{Label: "failed-thing", Percent: 0.3, Status: ProgressError},
	}

	for _, width := range []int{80, 40, 20, 10, 1} {
		got := MultiProgress(items, width, theme.DarkTheme())
		for i, l := range strings.Split(got, "\n") {
			if w := ansi.Width(l); w > width {
				t.Errorf("width=%d row %d: ansi.Width=%d exceeds width %d (line %q)", width, i, w, width, l)
			}
		}
	}
}
