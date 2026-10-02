package widgets

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

func TestProgressBar(t *testing.T) {
	tests := []struct {
		name    string
		percent float64
		width   int
		filled  int
	}{
		{"zero", 0, 10, 0},
		{"full", 1, 10, 10},
		{"half", 0.5, 10, 5},
		{"below zero clamps", -0.5, 10, 0},
		{"above one clamps", 1.5, 10, 10},
		{"rounds to nearest", 0.44, 10, 4}, // 4.4 -> 4
		{"rounds up at .5", 0.45, 10, 5},   // 4.5 -> 5 (round-half-up)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ProgressBar(tt.percent, tt.width, theme.DarkTheme())
			want := ansi.NewStyle().Foreground(theme.DarkTheme().Primary).Render(strings.Repeat("█", tt.filled)) +
				ansi.NewStyle().Foreground(theme.DarkTheme().Muted).Render(strings.Repeat("░", tt.width-tt.filled))
			if got != want {
				t.Errorf("ProgressBar(%v, %d, Dark) = %q, want %q", tt.percent, tt.width, got, want)
			}
		})
	}
}

func TestProgressBarZeroWidth(t *testing.T) {
	if got := ProgressBar(0.5, 0, theme.DarkTheme()); got != "" {
		t.Errorf("ProgressBar with width 0 = %q, want empty", got)
	}
}

func TestProgressBarWidthAware(t *testing.T) {
	got := ProgressBar(0.5, 10, theme.DarkTheme())
	if w := ansi.Width(got); w != 10 {
		t.Errorf("Width(ProgressBar(...)) = %d, want 10", w)
	}
}
