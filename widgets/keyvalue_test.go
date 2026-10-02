package widgets

import (
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

func TestKeyValue(t *testing.T) {
	tt := theme.DarkTheme()
	keyStyle := ansi.NewStyle().Foreground(tt.Muted).Bold()

	tests := []struct {
		name  string
		pairs []KV
		want  string
	}{
		{
			name:  "one pair per line as key: value",
			pairs: []KV{{"Name", "acline"}, {"Status", "running"}},
			want: keyStyle.Render("Name:") + "   acline\n" +
				keyStyle.Render("Status:") + " running",
		},
		{
			name:  "single pair has no stray padding",
			pairs: []KV{{"Name", "acline"}},
			want:  keyStyle.Render("Name:") + " acline",
		},
		{
			name:  "empty pairs returns empty string",
			pairs: nil,
			want:  "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := KeyValue(test.pairs, tt)
			if got != test.want {
				t.Errorf("KeyValue(%v, Dark) = %q, want %q", test.pairs, got, test.want)
			}
		})
	}
}

func TestKeyValueEmptyDoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("KeyValue panicked on empty pairs: %v", r)
		}
	}()

	if got := KeyValue([]KV{}, theme.DarkTheme()); got != "" {
		t.Errorf("KeyValue([], Dark) = %q, want empty string", got)
	}
	if got := KeyValue(nil, theme.DarkTheme()); got != "" {
		t.Errorf("KeyValue(nil, Dark) = %q, want empty string", got)
	}
}

// TestKeyValueAlignsValuesAcrossKeyWidths verifies that every value starts
// at the same rendered column regardless of key width, measuring with
// ansi.Width (via StripANSI) rather than len/byte count.
func TestKeyValueAlignsValuesAcrossKeyWidths(t *testing.T) {
	pairs := []KV{
		{"ID", "1"},
		{"Description", "a longer field"},
		{"On", "true"},
	}

	got := KeyValue(pairs, theme.DarkTheme())

	lines := splitLines(got)
	if len(lines) != len(pairs) {
		t.Fatalf("KeyValue produced %d lines, want %d", len(lines), len(pairs))
	}

	valueCol := -1
	for i, line := range lines {
		plain := ansi.StripANSI(line)
		wantSuffix := " " + pairs[i].Value
		if ansi.Width(plain) < ansi.Width(wantSuffix) || plain[len(plain)-len(wantSuffix):] != wantSuffix {
			t.Fatalf("line %d = %q, want it to end with %q", i, plain, wantSuffix)
		}
		col := ansi.Width(plain) - ansi.Width(pairs[i].Value)
		if valueCol == -1 {
			valueCol = col
		} else if col != valueCol {
			t.Errorf("line %d: value starts at column %d, want %d (same as other lines)", i, col, valueCol)
		}
	}
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	lines = append(lines, s[start:])
	return lines
}
