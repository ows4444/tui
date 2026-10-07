package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/input"
)

// fixture is a model on a fixed day and a directory with known contents, so
// what it draws does not depend on today or on where the test runs.
func fixture(t testing.TB) model {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "attachments")
	if err := os.MkdirAll(filepath.Join(dir, "drafts"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"agenda.md", "budget.csv", "notes.txt", filepath.Join("drafts", "old.md")} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return newModel(time.Date(2026, time.August, 14, 0, 0, 0, 0, time.UTC), dir)
}

// drive passes each message to Update and, as the Program would, feeds the
// message any returned Cmd produces back in. It reports whether a Cmd quit.
func drive(t *testing.T, m model, msgs ...tui.Msg) (model, bool) {
	t.Helper()
	quit := false
	for len(msgs) > 0 {
		msg := msgs[0]
		msgs = msgs[1:]
		next, cmd := m.Update(msg)
		m = next.(model)
		if cmd == nil {
			continue
		}
		switch out := cmd().(type) {
		case nil:
		case tui.QuitMsg:
			quit = true
		default:
			msgs = append([]tui.Msg{out}, msgs...)
		}
	}
	return m, quit
}

func key(t tui.KeyType) tui.Key { return tui.Key{Type: t} }

func char(r rune) tui.Key { return tui.Key{Type: tui.KeyRunes, Text: string(r), Code: r} }

var shiftTab = tui.Key{Type: tui.KeyTab, Mod: input.ModShift}

func TestConfirmingADateRecordsItAndMovesOn(t *testing.T) {
	m, _ := drive(t, fixture(t), key(tui.KeyRight), key(tui.KeyDown), key(tui.KeyEnter))
	if got := m.chosen[sectionDate]; got != "Sat 22 Aug 2026" {
		t.Fatalf("date = %q, want %q", got, "Sat 22 Aug 2026")
	}
	if m.focus != sectionColour {
		t.Fatalf("focus = %d, want the colour section", m.focus)
	}
}

func TestConfirmingAColourNamesIt(t *testing.T) {
	m, _ := drive(t, fixture(t), key(tui.KeyTab), key(tui.KeyRight), key(tui.KeyRight))
	if out := ansi.StripANSI(m.View()); !strings.Contains(out, "Cursor on: green") {
		t.Fatalf("the view does not name the swatch under the cursor:\n%s", out)
	}
	m, _ = drive(t, m, key(tui.KeyEnter))
	if got := m.chosen[sectionColour]; got != "green" {
		t.Fatalf("colour = %q, want %q", got, "green")
	}
	if m.focus != sectionFile {
		t.Fatalf("focus = %d, want the file section", m.focus)
	}
}

// Tab belongs to the sections here, so "/" is what switches the colour picker
// to its hex field, and a typed value comes back as that colour.
func TestHexFieldIsReachedWithSlash(t *testing.T) {
	m, _ := drive(t, fixture(t), key(tui.KeyTab), char('/'))
	if !m.colour.HexFocused() {
		t.Fatal(`"/" did not move to the hex field`)
	}
	m, _ = drive(t, m, char('#'), char('1'), char('2'), char('a'), char('b'), char('e'), char('f'), key(tui.KeyEnter))
	if got := m.chosen[sectionColour]; got != "#12abef" {
		t.Fatalf("colour = %q, want %q", got, "#12abef")
	}
}

func TestFilePickerDescendsAndConfirmsAFile(t *testing.T) {
	m, _ := drive(t, fixture(t), shiftTab)
	if m.focus != sectionFile {
		t.Fatalf("shift+tab from the first section: focus = %d, want the last", m.focus)
	}
	out := ansi.StripANSI(m.View())
	for _, want := range []string{"In: attachments", "drafts", "agenda.md"} {
		if !strings.Contains(out, want) {
			t.Fatalf("the file section does not show %q:\n%s", want, out)
		}
	}
	// Enter on a directory goes into it and confirms nothing.
	m, _ = drive(t, m, key(tui.KeyEnter))
	if m.chosen[sectionFile] != "" || filepath.Base(m.file.Dir) != "drafts" {
		t.Fatalf("after Enter on a directory: chosen=%q dir=%q", m.chosen[sectionFile], m.file.Dir)
	}
	m, _ = drive(t, m, key(tui.KeyEnter))
	if got := m.chosen[sectionFile]; got != "old.md" {
		t.Fatalf("file = %q, want %q", got, "old.md")
	}
}

func TestKeysGoOnlyToTheFocusedSection(t *testing.T) {
	before := fixture(t)
	m, _ := drive(t, before, key(tui.KeyTab), key(tui.KeyRight))
	if !m.date.Cursor().Equal(before.date.Cursor()) {
		t.Fatal("a key pressed in the colour section moved the date")
	}
}

func TestEscQuits(t *testing.T) {
	if _, quit := drive(t, fixture(t), key(tui.KeyEsc)); !quit {
		t.Fatal("esc did not quit")
	}
}

// Every section fits a 40x10 terminal, the smallest the size matrix draws.
func TestEverySectionFitsASmallTerminal(t *testing.T) {
	m, _ := drive(t, fixture(t), tui.ResizeMsg{Width: 40, Height: 10})
	for range sectionCount {
		lines := strings.Split(ansi.StripANSI(m.View()), "\n")
		if len(lines) > 10 {
			t.Errorf("section %d: %d rows in a 10-row terminal:\n%s", m.focus, len(lines), strings.Join(lines, "\n"))
		}
		for _, l := range lines {
			if w := ansi.Width(l); w > 40 {
				t.Errorf("section %d: a line is %d wide in a 40-column terminal: %q", m.focus, w, l)
			}
		}
		m, _ = drive(t, m, key(tui.KeyTab))
	}
}
