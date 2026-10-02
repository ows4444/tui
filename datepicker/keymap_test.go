package datepicker

import (
	"testing"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/keymap"
)

func TestBindingsListsActiveBindingsWithDesc(t *testing.T) {
	bs := New(date(2024, time.March, 15)).Bindings()
	if len(bs) != 7 {
		t.Fatalf("Bindings() has %d entries, want 7", len(bs))
	}
	for _, b := range bs {
		if b.Desc == "" || len(b.Keys) == 0 {
			t.Errorf("binding %+v lacks Desc or Keys", b)
		}
	}
}

func TestZeroKeyMapFallsBackToDefaults(t *testing.T) {
	m := Model{cursor: date(2024, time.March, 15)}
	m = sendKey(t, m, tui.KeyRight)
	if !sameDay(m.Cursor(), date(2024, time.March, 16)) {
		t.Fatalf("Cursor() = %v, want 2024-03-16 with zero KeyMap", m.Cursor())
	}
}

func TestRebindNextDayChangesBehaviour(t *testing.T) {
	m := New(date(2024, time.March, 15))
	m.KeyMap.NextDay = keymap.NewBinding("next day", "l")
	m = sendKey(t, m, tui.KeyRight)
	if !sameDay(m.Cursor(), date(2024, time.March, 15)) {
		t.Fatalf("old key still moves: %v", m.Cursor())
	}
	m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: "l"})
	if !sameDay(m.Cursor(), date(2024, time.March, 16)) {
		t.Fatalf("rebound key: Cursor() = %v, want 2024-03-16", m.Cursor())
	}
}
