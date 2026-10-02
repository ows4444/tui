package autocomplete

import (
	"testing"

	"github.com/ows4444/tui"
)

// Criterion: a copy keeps its Value() and Cursor() after the original
// receives any edit key or paste.
func TestCopyIsolation(t *testing.T) {
	msgs := map[string]tui.Msg{
		"backspace": tui.Key{Type: tui.KeyBackspace},
		"delete":    tui.Key{Type: tui.KeyDelete},
		"rune":      tui.Key{Type: tui.KeyRunes, Text: "1"},
		"ctrl+u":    tui.Key{Type: tui.KeyCtrl, Code: 'u'},
		"ctrl+k":    tui.Key{Type: tui.KeyCtrl, Code: 'k'},
		"ctrl+w":    tui.Key{Type: tui.KeyCtrl, Code: 'w'},
		"paste":     tui.PasteEvent{Text: "12"},
	}
	for name, msg := range msgs {
		m := New("a")
		m.Input.Focus()
		m.Input.SetValue("1234 5678")
		m.Input.SetCursor(4)
		snap := m
		wantV, wantC := snap.Input.Value(), snap.Input.Cursor()
		m, _ = m.Update(msg)
		if snap.Input.Value() != wantV || snap.Input.Cursor() != wantC {
			t.Errorf("%s: copy changed to %q cursor %d, want %q cursor %d", name, snap.Input.Value(), snap.Input.Cursor(), wantV, wantC)
		}
	}
}
