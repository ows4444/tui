package confirm

import (
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// Criterion #2 (spec #32): the highlighted option uses Theme.ResolvedStates().Selected.
func TestHighlightUsesSelectedState(t *testing.T) {
	m := New("ok?")
	custom := ansi.NewStyle().Underline()
	m.Theme.States.Selected = custom
	want := custom.Bold().Render("[Yes]")
	if got := m.options(); got[:len(want)] != want {
		t.Fatalf("options = %q, want prefix %q", got, want)
	}

	m = New("ok?").SetTheme(theme.LightTheme())
	want = theme.LightTheme().ResolvedStates().Selected.Bold().Render("[Yes]")
	if got := m.options(); got[:len(want)] != want {
		t.Fatalf("default states: options = %q, want prefix %q", got, want)
	}
}
