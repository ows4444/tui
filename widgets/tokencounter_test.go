package widgets

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

func TestTokenCounterCompactCounts(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{-5, "0"}, {0, "0"}, {7, "7"}, {999, "999"},
		{1000, "1k"}, {1049, "1k"}, {1050, "1.1k"}, {1234, "1.2k"}, {1250, "1.3k"},
		{8000, "8k"}, {99_949, "99.9k"}, {100_000, "100k"}, {999_949, "999.9k"},
		{999_950, "1M"}, {1_000_000, "1M"}, {1_500_000, "1.5M"}, {12_340_000, "12.3M"},
		{999_950_000, "1B"}, {2_500_000_000, "2.5B"},
	}
	for _, tt := range tests {
		got := ansi.StripANSI(TokenCounter(tt.n, 0, theme.DarkTheme()))
		if want := tt.want + " tokens"; got != want {
			t.Errorf("TokenCounter(%d, 0) = %q, want %q", tt.n, got, want)
		}
	}
}

func TestTokenCounterReadout(t *testing.T) {
	tests := []struct {
		used, limit int
		want        string
	}{
		{1234, 8000, "1.2k / 8k tokens"},
		{0, 200_000, "0 / 200k tokens"},
		{950, 1000, "950 / 1k tokens"},
		{5, 0, "5 tokens"},
		{5, -1, "5 tokens"},
		{-3, 100, "0 / 100 tokens"},
	}
	for _, tt := range tests {
		got := ansi.StripANSI(TokenCounter(tt.used, tt.limit, theme.DarkTheme()))
		if got != tt.want {
			t.Errorf("TokenCounter(%d, %d) = %q, want %q", tt.used, tt.limit, got, tt.want)
		}
	}
}

func TestTokenCounterColourThresholds(t *testing.T) {
	th := theme.DarkTheme()
	muted := func(s string) string { return ansi.NewStyle().Foreground(th.Muted).Render(s) }
	warn := func(s string) string { return ansi.NewStyle().Foreground(th.Warning).Render(s) }
	errc := func(s string) string { return ansi.NewStyle().Bold().Foreground(th.Error).Render(s) }

	tests := []struct {
		name        string
		used, limit int
		style       func(string) string
	}{
		{"empty", 0, 100, muted},
		{"just under 80%", 79, 100, muted},
		{"exactly 80%", 80, 100, warn},
		{"just under 100%", 99, 100, warn},
		{"exactly 100%", 100, 100, errc},
		{"over the limit", 150, 100, errc},
		{"no limit", 1_000_000, 0, muted},
		{"tiny limit boundary", 4, 5, warn}, // 80% of 5, integer maths
		{"huge values don't overflow", 3_200_000_000_000, 4_000_000_000_000, warn},
	}
	for _, tt := range tests {
		got := TokenCounter(tt.used, tt.limit, th)
		want := tt.style(ansi.StripANSI(got))
		if got != want {
			t.Errorf("%s: TokenCounter(%d, %d) = %q, want %q", tt.name, tt.used, tt.limit, got, want)
		}
	}
}

func TestTokenCounterSingleLineAndTheme(t *testing.T) {
	got := TokenCounter(1234, 8000, theme.LightTheme())
	if strings.Contains(got, "\n") {
		t.Errorf("multi-line output: %q", got)
	}
	if w, want := ansi.Width(got), len("1.2k / 8k tokens"); w != want {
		t.Errorf("Width = %d, want %d", w, want)
	}
	want := ansi.NewStyle().Foreground(theme.LightTheme().Muted).Render("1.2k / 8k tokens")
	if got != want {
		t.Errorf("should use the given theme's colours: got %q, want %q", got, want)
	}
}
