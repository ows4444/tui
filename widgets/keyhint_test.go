package widgets

import (
	"testing"

	"github.com/ows4444/tui/keymap"
)

func TestKeyHint(t *testing.T) {
	got := KeyHint("enter", "submit")
	want := keyHintDimStyle.Render("[") + keyHintKeyStyle.Render("enter") + keyHintDimStyle.Render("] submit")
	if got != want {
		t.Errorf("KeyHint(%q, %q) = %q, want %q", "enter", "submit", got, want)
	}
}

func TestKeyHints(t *testing.T) {
	got := KeyHints("  ", Hint{"up/down", "move"}, Hint{"q", "quit"})
	want := KeyHint("up/down", "move") + "  " + KeyHint("q", "quit")
	if got != want {
		t.Errorf("KeyHints() = %q, want %q", got, want)
	}
}

func TestKeyHintsEmpty(t *testing.T) {
	if got := KeyHints("  "); got != "" {
		t.Errorf("KeyHints() with no hints = %q, want empty", got)
	}
}

func TestKeyHintsSingle(t *testing.T) {
	got := KeyHints("  ", Hint{"q", "quit"})
	want := KeyHint("q", "quit")
	if got != want {
		t.Errorf("KeyHints() with one hint = %q, want %q (no trailing separator)", got, want)
	}
}

// TestHintsFromKeymap proves criterion #93 for the key-hint widget: a binding
// registered once in keymap shows up in KeyHints with no second list.
func TestHintsFromKeymap(t *testing.T) {
	var r keymap.Registry
	r.Add(keymap.Binding{Keys: []string{"q"}, Desc: "quit"})
	got := KeyHints("  ", HintsFromKeymap(&r, "")...)
	if want := KeyHint("q", "quit"); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
