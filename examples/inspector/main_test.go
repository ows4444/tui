package main

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
)

func isQuit(cmd tui.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tui.QuitMsg)
	return ok
}

func send(m model, k tui.Key) model {
	next, _ := m.Update(k)
	return next.(model)
}

func TestQuitKeys(t *testing.T) {
	for name, k := range map[string]tui.Key{
		"ctrl+c": {Type: tui.KeyCtrlC},
		"esc":    {Type: tui.KeyEsc},
		"q":      {Type: tui.KeyRunes, Text: "q"},
	} {
		if _, cmd := initialModel().Update(k); !isQuit(cmd) {
			t.Errorf("%s did not quit", name)
		}
	}
}

func TestNonKeyMessagesAreIgnored(t *testing.T) {
	if _, cmd := initialModel().Update(tui.ResizeMsg{Width: 80, Height: 24}); cmd != nil {
		t.Error("a ResizeMsg returned a Cmd")
	}
}

func TestTableTabShowsRowsAndDownMovesCursor(t *testing.T) {
	m := initialModel()
	out := ansi.StripANSI(m.View())
	for _, want := range []string{"Table", "JSON", "api-1", "degraded"} {
		if !strings.Contains(out, want) {
			t.Errorf("table tab missing %q:\n%s", want, out)
		}
	}
	m = send(m, tui.Key{Type: tui.KeyDown})
	if m.table.Cursor() != 1 {
		t.Errorf("table cursor = %d after Down, want 1", m.table.Cursor())
	}
}

func TestRightSwitchesToTheJSONTab(t *testing.T) {
	m := send(initialModel(), tui.Key{Type: tui.KeyRight})
	if m.tabs.Active() != 1 {
		t.Fatalf("active tab = %d after Right, want 1", m.tabs.Active())
	}
	out := ansi.StripANSI(m.View())
	if !strings.Contains(out, "root") || strings.Contains(out, "api-1") {
		t.Errorf("JSON tab should show the tree, not the table:\n%s", out)
	}
	// Enter expands the root node, revealing the sample's keys.
	if exp := ansi.StripANSI(send(m, tui.Key{Type: tui.KeyEnter}).View()); !strings.Contains(exp, "service") {
		t.Errorf("Enter did not expand the tree:\n%s", exp)
	}
	// Keys now go to the tree, not the table.
	before := m.table.Cursor()
	m = send(m, tui.Key{Type: tui.KeyDown})
	if m.table.Cursor() != before {
		t.Error("Down on the JSON tab moved the table cursor")
	}
	if m = send(m, tui.Key{Type: tui.KeyLeft}); m.tabs.Active() != 0 {
		t.Errorf("Left did not return to the Table tab, active=%d", m.tabs.Active())
	}
}
