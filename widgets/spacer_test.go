package widgets

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
)

func TestSpacer(t *testing.T) {
	got := Spacer(4, 3)
	lines := strings.Split(got, "\n")
	if len(lines) != 3 {
		t.Fatalf("Spacer(4, 3) has %d lines, want 3", len(lines))
	}
	for i, line := range lines {
		if w := ansi.Width(line); w != 4 {
			t.Errorf("Spacer(4, 3) line %d width = %d, want 4", i, w)
		}
		if strings.TrimSpace(line) != "" {
			t.Errorf("Spacer(4, 3) line %d = %q, want blank", i, line)
		}
	}
}

func TestSpacerNonPositiveReturnsEmpty(t *testing.T) {
	tests := []struct {
		width, height int
	}{
		{0, 5},
		{-1, 5},
		{5, 0},
		{5, -1},
		{0, 0},
		{-1, -1},
	}
	for _, tt := range tests {
		if got := Spacer(tt.width, tt.height); got != "" {
			t.Errorf("Spacer(%d, %d) = %q, want empty string", tt.width, tt.height, got)
		}
	}
}
