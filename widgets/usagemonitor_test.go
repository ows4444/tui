package widgets

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// #522: values are formatted via CompactCount and colored via the same
// threshold rule TokenCounter uses.
func TestUsageMonitorFormattingAndColour(t *testing.T) {
	th := theme.DarkTheme()
	muted := func(s string) string { return ansi.NewStyle().Foreground(th.Muted).Render(s) }
	warn := func(s string) string { return ansi.NewStyle().Foreground(th.Warning).Render(s) }
	errc := func(s string) string { return ansi.NewStyle().Bold().Foreground(th.Error).Render(s) }

	tests := []struct {
		name       string
		value, cap int
		wantText   string
		wantStyled func(string) string
	}{
		{"under 80%, exact count", 5, 100, "5 / 100", muted},
		{"under 80%, compact count", 1234, 8000, "1.2k / 8k", muted},
		{"exactly 80%", 80, 100, "80 / 100", warn},
		{"at limit", 100, 100, "100 / 100", errc},
		{"over limit", 150, 100, "150 / 100", errc},
		{"no limit", 1_500_000, 0, "1.5M", muted},
	}
	for _, tt := range tests {
		got := UsageMonitor("Usage", []UsageStat{{Label: "Requests", Value: tt.value, Limit: tt.cap}}, th)
		want := "Usage\n" + ansi.NewStyle().Foreground(th.Muted).Bold().Render("Requests:") + " " + tt.wantStyled(tt.wantText)
		if got != want {
			t.Errorf("%s: got %q, want %q", tt.name, got, want)
		}
	}
}

// #523: multiple rows align labels to a common column via KeyValue's
// padding convention.
func TestUsageMonitorLabelAlignment(t *testing.T) {
	th := theme.DarkTheme()
	got := UsageMonitor("Usage", []UsageStat{
		{Label: "Requests", Value: 5, Limit: 100},
		{Label: "CPU", Value: 10, Limit: 0},
	}, th)

	lines := strings.Split(got, "\n")
	if len(lines) != 3 {
		t.Fatalf("expected title + 2 rows, got %d lines: %q", len(lines), got)
	}

	// Compare against KeyValue's own padding for the same labels, since
	// UsageMonitor must follow the same alignment convention: the
	// label+colon+padding prefix before the value has the same visible
	// width in both, regardless of value content.
	kv := KeyValue([]KV{
		{Key: "Requests", Value: "X"},
		{Key: "CPU", Value: "X"},
	}, th)
	kvLines := strings.Split(kv, "\n")
	labels := []string{"Requests", "CPU"}

	for i, line := range lines[1:] {
		wantPrefixWidth := ansi.Width(kvLines[i]) - ansi.Width("X")

		stripped := ansi.StripANSI(line)
		afterColon := strings.TrimPrefix(stripped, labels[i]+":")
		valueWidth := ansi.Width(strings.TrimLeft(afterColon, " "))
		gotPrefixWidth := ansi.Width(stripped) - valueWidth

		if gotPrefixWidth != wantPrefixWidth {
			t.Errorf("row %d: prefix width = %d, want %d (line=%q)", i, gotPrefixWidth, wantPrefixWidth, line)
		}
	}
}

// #524: limit <= 0 never gets warning/error coloring, always stays Muted.
func TestUsageMonitorNoLimitStaysMuted(t *testing.T) {
	th := theme.DarkTheme()
	tests := []int{0, -1, -100}
	for _, limit := range tests {
		got := UsageMonitor("", []UsageStat{{Label: "Tokens", Value: 1_000_000_000, Limit: limit}}, th)
		want := ansi.NewStyle().Foreground(th.Muted).Bold().Render("Tokens:") + " " + ansi.NewStyle().Foreground(th.Muted).Render("1B")
		if got != want {
			t.Errorf("limit=%d: got %q, want %q", limit, got, want)
		}
	}
}

// #525: empty stats list renders without panicking, just the title (or ""
// if the title is also empty).
func TestUsageMonitorEmptyStats(t *testing.T) {
	if got := UsageMonitor("Usage", nil, theme.DarkTheme()); got != "Usage" {
		t.Errorf("with title: got %q, want %q", got, "Usage")
	}
	if got := UsageMonitor("", nil, theme.DarkTheme()); got != "" {
		t.Errorf("without title: got %q, want %q", got, "")
	}
	if got := UsageMonitor("Usage", []UsageStat{}, theme.DarkTheme()); got != "Usage" {
		t.Errorf("with empty (non-nil) slice: got %q, want %q", got, "Usage")
	}
}

func TestUsageMonitorSingleLinePerRow(t *testing.T) {
	got := UsageMonitor("Usage", []UsageStat{
		{Label: "Requests", Value: 5, Limit: 100},
	}, theme.DarkTheme())
	for _, line := range strings.Split(got, "\n") {
		if strings.Contains(line, "\n") {
			t.Errorf("unexpected embedded newline in %q", line)
		}
	}
}
