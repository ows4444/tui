package passwordinput

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
		m := New()
		m.Focus()
		m.SetValue("1234 5678")
		m.SetCursor(4)
		snap := m
		wantV, wantC := snap.Value(), snap.Cursor()
		m, _ = m.Update(msg)
		if snap.Value() != wantV || snap.Cursor() != wantC {
			t.Errorf("%s: copy changed to %q cursor %d, want %q cursor %d", name, snap.Value(), snap.Cursor(), wantV, wantC)
		}
	}
}
