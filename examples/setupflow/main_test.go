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

// View composes BigText, Alert and picker.Model as a numbered-select
// menu of setup steps (criterion #611).
func TestViewComposesTitleAlertAndMenu(t *testing.T) {
	m := initialModel()
	view := ansi.StripANSI(m.View())

	if !strings.Contains(view, "No step applied yet.") {
		t.Errorf("View should render the Alert status, got:\n%s", view)
	}
	if !strings.Contains(view, "Configure workspace") {
		t.Errorf("View should render the picker menu, got:\n%s", view)
	}
}

func TestSelectingStepUpdatesStatus(t *testing.T) {
	m := initialModel()
	m = selectMenuItem(t, m, "Configure notifications")

	if m.applied != "Configure notifications" {
		t.Fatalf("applied = %q, want %q", m.applied, "Configure notifications")
	}
	view := ansi.StripANSI(m.View())
	if !strings.Contains(view, "Applied: Configure notifications") {
		t.Errorf("View after selection should show the applied step, got:\n%s", view)
	}
}

func TestCtrlCQuits(t *testing.T) {
	m := initialModel()
	_, cmd := m.Update(tui.Key{Type: tui.KeyCtrlC})
	if cmd == nil {
		t.Fatal("Ctrl+C should return a quit Cmd")
	}
	if _, ok := cmd().(tui.QuitMsg); !ok {
		t.Fatalf("Ctrl+C Cmd delivered %T, want tui.QuitMsg", cmd())
	}
}

func TestDocCommentStatesNoNewPattern(t *testing.T) {
	src, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("reading main.go: %v", err)
	}
	text := string(src)
	if !strings.Contains(text, "no new pattern") {
		t.Error("doc comment should state this introduces no new pattern")
	}
}
