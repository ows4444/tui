package filepicker

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ows4444/tui"
)

func typed(m Model, s string) Model {
	for _, r := range s {
		m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: string(r)})
	}
	return m
}

func dirWith(t *testing.T, names ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, n := range names {
		if err := os.WriteFile(filepath.Join(dir, n), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestTypeAheadJumpsToTheEntryThatStartsWithTheLetters(t *testing.T) {
	m := New(dirWith(t, "alpha.go", "beta.go", "Bravo.txt", "gamma.go"))
	m.TypeAhead = true
	name := func() string { return m.entries[m.Cursor()].Name }
	if m = typed(m, "g"); name() != "gamma.go" {
		t.Errorf("g -> %q", name())
	}
	m.ta.Clear()
	m.cursor = 0
	if m = typed(m, "br"); name() != "Bravo.txt" {
		t.Errorf("br -> %q (case must not matter)", name())
	}
}

func TestTypeAheadOffByDefaultAndBindingsWin(t *testing.T) {
	m := New(dirWith(t, "alpha", "beta"))
	if m = typed(m, "b"); m.Cursor() != 0 {
		t.Error("letters moved the cursor with TypeAhead off")
	}
	m.TypeAhead = true
	m.KeyMap = DefaultKeyMap()
	m.KeyMap.Down.Keys = append([]string{"b"}, m.KeyMap.Down.Keys...)
	if m = typed(m, "b"); m.Cursor() != 1 {
		t.Errorf("a bound letter was taken for type-ahead: cursor %d", m.Cursor())
	}
	empty := New(t.TempDir())
	empty.TypeAhead = true
	_ = typed(empty, "x") // no entries: no panic
}
