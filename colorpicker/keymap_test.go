package colorpicker

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/keymap"
)

func testPalette() []ansi.Color {
	return []ansi.Color{ansi.RGB{R: 1}, ansi.RGB{G: 1}, ansi.RGB{B: 1}}
}

func TestBindingsFollowFocusedPane(t *testing.T) {
	m := New(testPalette()...)
	pal := m.Bindings()
	m, _ = m.Update(tui.Key{Type: tui.KeyTab})
	hex := m.Bindings()
	if len(pal) != 4 || len(hex) <= 2 {
		t.Fatalf("binding counts palette=%d hex=%d", len(pal), len(hex))
	}
	for _, b := range append(pal, hex...) {
		if b.Desc == "" || len(b.Keys) == 0 {
			t.Errorf("binding %+v lacks Desc or Keys", b)
		}
	}
}

func TestZeroKeyMapFallsBackToDefaults(t *testing.T) {
	m := Model{Palette: testPalette()}
	m, _ = m.Update(tui.Key{Type: tui.KeyRight})
	if m.Cursor() != 1 {
		t.Fatalf("Cursor() = %d, want 1 with zero KeyMap", m.Cursor())
	}
}

func TestRebindNextChangesBehaviour(t *testing.T) {
	m := New(testPalette()...)
	m.KeyMap.Next = keymap.NewBinding("next", "l")
	m, _ = m.Update(tui.Key{Type: tui.KeyRight})
	if m.Cursor() != 0 {
		t.Fatalf("old key still moves: Cursor() = %d", m.Cursor())
	}
	m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: "l"})
	if m.Cursor() != 1 {
		t.Fatalf("rebound key: Cursor() = %d, want 1", m.Cursor())
	}
}
