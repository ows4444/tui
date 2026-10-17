package form

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
)

func typeRunes(m Model, s string) Model {
	for _, r := range s {
		m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: string(r)})
	}
	return m
}

func ctrlKey(m Model, r rune) (Model, tui.Cmd) {
	return m.Update(tui.Key{Type: tui.KeyCtrl, Code: r})
}

func TestConfirmChoosesWithYNArrowsAndSpace(t *testing.T) {
	m := New(Field{Name: "ok", Label: "Delete?", Kind: FieldConfirm})
	m.Focus()
	if got := m.Values()["ok"]; got != "false" {
		t.Fatalf("starts on %q; want false", got)
	}
	if v := ansi.StripANSI(m.View()); v != "Delete?:  Yes  [No]" {
		t.Fatalf("view %q", v)
	}
	m = typeRunes(m, "y")
	if !Bool(m, "ok") || ansi.StripANSI(m.View()) != "Delete?: [Yes]  No " {
		t.Fatalf("y did not choose Yes: %q", m.View())
	}
	m = typeRunes(m, "N")
	if Bool(m, "ok") {
		t.Fatal("N did not choose No")
	}
	m = key(m, tui.KeyLeft)
	if !Bool(m, "ok") {
		t.Fatal("Left should choose Yes")
	}
	m = key(m, tui.KeyRight)
	if Bool(m, "ok") {
		t.Fatal("Right should choose No")
	}
	m = space(m)
	if !Bool(m, "ok") {
		t.Fatal("Space did not toggle")
	}
	m = typeRunes(m, "q") // other keys change nothing
	if !Bool(m, "ok") {
		t.Fatal("an unrelated key changed the choice")
	}
}

func TestConfirmInitialValueMouseAndAccessibility(t *testing.T) {
	m := New(Field{Name: "ok", Label: "Sure", Kind: FieldConfirm, Value: "TRUE"})
	m.Focus()
	if !Bool(m, "ok") {
		t.Fatal("Value true did not start on Yes")
	}
	if got := m.Linearize(); !strings.Contains(got, "Sure, confirm, focused, yes") {
		t.Fatalf("linearize %q", got)
	}
	m.Mouse = true
	m.Bounds.W, m.Bounds.H = 40, 3
	m, _ = m.Update(tui.MouseEvent{X: 1, Y: 0, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	if Bool(m, "ok") {
		t.Fatal("a click did not toggle")
	}
	if len(m.Bindings()) < 5 {
		t.Fatal("confirm lends no bindings")
	}
}

func TestNoteIsNotAFieldOrAFocusStop(t *testing.T) {
	m := New(
		Field{Kind: FieldNote, Label: "Everything below is optional."},
		Field{Name: "a", Label: "A"},
		Field{Kind: FieldNote, Label: "Almost there."},
		Field{Name: "b", Label: "B"},
	)
	m.Focus()
	if m.Current() != "a" {
		t.Fatalf("focus starts on %q; want the first real field", m.Current())
	}
	m = key(m, tui.KeyTab)
	if m.Current() != "b" {
		t.Fatalf("Tab landed on %q; want it to skip the note", m.Current())
	}
	m = key(m, tui.KeyTab)
	if m.Current() != "a" {
		t.Fatalf("Tab wrapped to %q", m.Current())
	}
	if vals := m.Values(); len(vals) != 2 {
		t.Fatalf("Values %v; a note has no value", vals)
	}
	v := ansi.StripANSI(m.View())
	if !strings.Contains(v, "Everything below is optional.") || !strings.Contains(v, "Almost there.") {
		t.Fatalf("notes not drawn:\n%s", v)
	}
	if !strings.Contains(m.Linearize(), "note, Everything below is optional.") {
		t.Fatalf("linearize %q", m.Linearize())
	}
	m, cmd := m.Submit()
	if cmd == nil {
		t.Fatal("a form with notes could not submit")
	}
	if sub, ok := cmd().(SubmittedMsg); !ok || len(sub.Values) != 2 {
		t.Fatalf("submitted %+v", sub)
	}
}

func TestNoteInGroupsStaysOutOfTheRing(t *testing.T) {
	m := NewGroups(
		Group{Title: "One", Fields: []Field{{Kind: FieldNote, Label: "hello"}, {Name: "a", Label: "A"}}},
		Group{Title: "Two", Fields: []Field{{Name: "b", Label: "B"}, {Kind: FieldNote, Label: "bye"}}},
	)
	m.Focus()
	if m.Current() != "a" {
		t.Fatalf("group one starts on %q", m.Current())
	}
	m = key(m, tui.KeyEnter)
	if m.Current() != "b" || m.CurrentGroup() != 1 {
		t.Fatalf("Enter moved to %q in group %d", m.Current(), m.CurrentGroup())
	}
	m = key(m, tui.KeyTab)
	if m.Current() != "b" {
		t.Fatalf("Tab in group two went to %q", m.Current())
	}
}

func TestOnlyNotes(t *testing.T) {
	m := New(Field{Kind: FieldNote, Label: "just a note"})
	m.Focus()
	if m.Current() != "" || len(m.Values()) != 0 {
		t.Fatal("a form of notes has a field")
	}
	_ = m.View()
}

func TestFileFieldBrowsesAndPicks(t *testing.T) {
	dir := t.TempDir()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(os.WriteFile(filepath.Join(dir, "a.go"), nil, 0o644))
	must(os.WriteFile(filepath.Join(dir, "b.txt"), nil, 0o644))
	m := New(
		Field{Name: "src", Label: "Source", Kind: FieldFile, Value: dir, Options: []string{".go"}, Validators: []Validator{Required(), FileExists()}},
		Field{Name: "other", Label: "Other"},
	)
	m.Focus()
	m, _ = ctrlKey(m, 'o')
	view := ansi.StripANSI(m.View())
	if !strings.Contains(view, "a.go") || strings.Contains(view, "b.txt") {
		t.Fatalf("the list should show the .go file only:\n%s", view)
	}
	if !strings.Contains(view, "Enter picks, Esc closes") {
		t.Fatalf("no hint:\n%s", view)
	}
	// Enter picks the file and does not submit the form.
	m, cmd := m.Update(tui.Key{Type: tui.KeyEnter})
	if cmd != nil {
		if _, submitted := cmd().(SubmittedMsg); submitted {
			t.Fatal("Enter in the list submitted the form")
		}
	}
	if got := m.Values()["src"]; got != filepath.Join(dir, "a.go") {
		t.Fatalf("path %q", got)
	}
	if strings.Contains(ansi.StripANSI(m.View()), "Esc closes") {
		t.Fatal("the list stayed open after a pick")
	}
	if err := FileExists()(m.Values()["src"]); err != nil {
		t.Fatalf("FileExists: %v", err)
	}
}

func TestFileFieldEscClosesWithoutLeaving(t *testing.T) {
	m := New(Field{Name: "p", Label: "Path", Kind: FieldFile, Value: t.TempDir()}, Field{Name: "q", Label: "Q"})
	m.Focus()
	m, _ = ctrlKey(m, 'o')
	if len(m.Bindings()) == 0 || !strings.Contains(m.Linearize(), "browsing") {
		t.Fatal("open list not announced")
	}
	m = key(m, tui.KeyTab) // swallowed by the list, focus stays
	if m.Current() != "p" {
		t.Fatalf("Tab left the open list: %q", m.Current())
	}
	m = key(m, tui.KeyEsc)
	if strings.Contains(m.Linearize(), "browsing") {
		t.Fatal("Esc did not close the list")
	}
	m = key(m, tui.KeyTab)
	if m.Current() != "q" {
		t.Fatal("Tab does not move on once the list is closed")
	}
}

func TestFileFieldTypesLikeText(t *testing.T) {
	m := New(Field{Name: "p", Label: "Path", Kind: FieldFile})
	m.Focus()
	m = typeRunes(m, "/tmp/x")
	if m.Values()["p"] != "/tmp/x" {
		t.Fatalf("typed %q", m.Values()["p"])
	}
	if m.LayoutNode() == nil {
		t.Fatal("no layout node")
	}
}

func TestFileExists(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "f")
	if err := os.WriteFile(f, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	v := FileExists()
	if v("") != nil || v(f) != nil {
		t.Fatal("rejected an empty value or a real file")
	}
	if v(filepath.Join(dir, "nope")) == nil || v(dir) == nil {
		t.Fatal("accepted a missing path or a directory")
	}
}

func TestCustomKeyMapWithoutBrowseGetsTheDefault(t *testing.T) {
	m := New(Field{Name: "p", Label: "P", Kind: FieldFile, Value: t.TempDir()})
	m.KeyMap = DefaultKeyMap()
	m.KeyMap.Browse.Keys = nil
	m.Focus()
	m, _ = ctrlKey(m, 'o')
	if !m.browsingFile() {
		t.Fatal("a KeyMap with no Browse key lost the default")
	}
}
