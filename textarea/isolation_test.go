package textarea

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
)

// Criterion: a copy of a textarea Model keeps its Value() after the original
// receives any edit key, Enter or paste.
func TestCopyIsolation(t *testing.T) {
	msgs := map[string]tui.Msg{
		"backspace":   tui.Key{Type: tui.KeyBackspace},
		"delete":      tui.Key{Type: tui.KeyDelete},
		"enter":       tui.Key{Type: tui.KeyEnter},
		"rune":        rk('X'),
		"ctrl+u":      ck('u'),
		"ctrl+k":      ck('k'),
		"ctrl+w":      ck('w'),
		"paste":       tui.PasteEvent{Text: "PASTE\nED"},
		"paste-short": tui.PasteEvent{Text: "P"},
	}
	for name, msg := range msgs {
		m := focused()
		m.SetValue("hello\nworld foo")
		m.SetCursor(9)
		snap := m
		m, _ = m.Update(msg)
		if got := snap.Value(); got != "hello\nworld foo" {
			t.Errorf("%s: copy Value() = %q", name, got)
		}
	}
}

func ck(r rune) tui.Key { return tui.Key{Type: tui.KeyCtrl, Code: r} }

// Criterion: a paste with escapes/C0 controls is stored without them.
func TestPasteSanitised(t *testing.T) {
	m := New()
	m.Focus()
	m, _ = m.Update(tui.PasteEvent{Text: "a\x1b]52;c;ZXZpbA==\x07b\x1b[31mc\td\x00\x08\ne"})
	if got, want := m.Value(), "abc     d\ne"; got != want {
		t.Errorf("Value() = %q, want %q", got, want)
	}
	if strings.Contains(m.View(), "\x1b]") {
		t.Errorf("View() renders OSC: %q", m.View())
	}
}
