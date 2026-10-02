package helpscreen

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/widgets"
)

func TestKeyUsesTypographyStrong(t *testing.T) {
	m := New(widgets.Hint{Key: "q", Action: "quit"})
	m.Theme.Typography.Strong = ansi.NewStyle().Foreground(ansi.RGB{R: 7, G: 7, B: 7})
	if !strings.Contains(m.Render(strings.Repeat(strings.Repeat("x", 30)+"\n", 9)), m.Theme.Typography.Strong.Render("q")) {
		t.Fatal("Render ignores Typography.Strong")
	}
}

func TestDefaultHintMatchesKeyHint(t *testing.T) {
	m := New()
	if got, want := m.hint(widgets.Hint{Key: "q", Action: "quit"}), widgets.KeyHint("q", "quit"); got != want {
		t.Fatalf("hint = %q, want %q", got, want)
	}
}
