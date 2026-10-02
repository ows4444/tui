package main

import (
	"os"
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/picker"
)

func selectMenuItem(t *testing.T, m model, label string) model {
	t.Helper()

	target := -1
	for i, item := range m.menu.Items {
		if item.Label == label {
			target = i
			break
		}
	}
	if target == -1 {
		t.Fatalf("no menu item labelled %q", label)
	}

	for m.menu.Cursor() != target {
		dir := tui.KeyDown
		if m.menu.Cursor() > target {
			dir = tui.KeyUp
		}
		next, _ := m.Update(tui.Key{Type: dir})
		m = next.(model)
	}

	next, cmd := m.Update(tui.Key{Type: tui.KeyEnter})
	m = next.(model)
	if cmd == nil {
		return m
	}
	msg := cmd()
	if _, ok := msg.(picker.SelectedMsg); !ok {
		return m
	}
	next, _ = m.Update(msg)
	return next.(model)
}

// View composes BigText, Banner and picker.Model as a numbered-select
// menu (criterion #610).
func TestViewComposesTitleBannerAndMenu(t *testing.T) {
	m := initialModel()
	view := ansi.StripANSI(m.View())

	if !strings.Contains(view, "Pick an account") {
		t.Errorf("View should render the Banner announcement, got:\n%s", view)
	}
	if !strings.Contains(view, "Continue as Ada Lovelace") {
		t.Errorf("View should render the picker menu, got:\n%s", view)
	}
}

// Selecting an account confirms it — with no credential check of any
// kind, per the "no auth logic" scope in criterion #610.
func TestSelectingAccountConfirmsWithoutAuth(t *testing.T) {
	m := initialModel()
	m = selectMenuItem(t, m, "Continue as guest")

	if !m.done {
		t.Fatal("selecting a menu item should mark the flow done")
	}
	if m.chosen != "Continue as guest" {
		t.Fatalf("chosen = %q, want %q", m.chosen, "Continue as guest")
	}

	view := ansi.StripANSI(m.View())
	if !strings.Contains(view, "Signed in as: Continue as guest") {
		t.Errorf("View after selection should confirm the choice, got:\n%s", view)
	}
}

func TestQuitFromMenu(t *testing.T) {
	m := initialModel()
	m = selectMenuItem(t, m, "Quit")
	// selectMenuItem returns early (cmd delivers QuitMsg, not SelectedMsg)
	// so m.done stays false; the real event loop is what acts on QuitMsg.
	if m.done {
		t.Fatal("Quit should not be treated as an account selection")
	}
}

func TestCtrlCQuitsFromMenu(t *testing.T) {
	m := initialModel()
	_, cmd := m.Update(tui.Key{Type: tui.KeyCtrlC})
	if cmd == nil {
		t.Fatal("Ctrl+C should return a quit Cmd")
	}
	if _, ok := cmd().(tui.QuitMsg); !ok {
		t.Fatalf("Ctrl+C Cmd delivered %T, want tui.QuitMsg", cmd())
	}
}

func TestDocCommentStatesNoAuthLogic(t *testing.T) {
	src, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("reading main.go: %v", err)
	}
	text := string(src)
	if !strings.Contains(text, "no authentication logic") {
		t.Error("doc comment should state there is no authentication logic")
	}
}
