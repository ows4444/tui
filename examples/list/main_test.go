package main

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/multiselect"
)

func isQuit(cmd tui.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tui.QuitMsg)
	return ok
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

func TestSpaceTogglesTheCursorItemAndStatsFollow(t *testing.T) {
	m := initialModel()
	next, _ := m.Update(tui.Key{Type: tui.KeySpace})
	m = next.(model)
	if got := len(m.list.SelectedIndexes()); got != 1 {
		t.Fatalf("selected = %d after space, want 1", got)
	}
	if out := ansi.StripANSI(m.View()); !strings.Contains(out, "1 / 6") {
		t.Errorf("stats box missing \"1 / 6\":\n%s", out)
	}
}

func TestMouseClickTogglesTheClickedRow(t *testing.T) {
	m := initialModel()
	click := tui.MouseEvent{Action: tui.MouseActionPress, Button: tui.MouseButtonLeft, Y: listTopOffset + 2}
	next, _ := m.Update(click)
	m = next.(model)
	if m.list.Cursor() != 2 || len(m.list.SelectedIndexes()) != 1 || m.list.SelectedIndexes()[0] != 2 {
		t.Errorf("cursor=%d selected=%v, want row 2 selected", m.list.Cursor(), m.list.SelectedIndexes())
	}

	// Clicks above the list, below it, or with another button change nothing.
	for _, e := range []tui.MouseEvent{
		{Action: tui.MouseActionPress, Button: tui.MouseButtonLeft, Y: 0},
		{Action: tui.MouseActionPress, Button: tui.MouseButtonLeft, Y: listTopOffset + 99},
		{Action: tui.MouseActionPress, Button: tui.MouseButtonRight, Y: listTopOffset},
	} {
		n, _ := m.Update(e)
		if got := len(n.(model).list.SelectedIndexes()); got != 1 {
			t.Errorf("click %+v changed the selection to %d items", e, got)
		}
	}
}

func TestConfirmedMsgShowsConfirmation(t *testing.T) {
	m := initialModel()
	if strings.Contains(ansi.StripANSI(m.View()), "Confirmed!") {
		t.Fatal("Confirmed! shown before confirming")
	}
	next, _ := m.Update(multiselect.ConfirmedMsg{})
	if out := ansi.StripANSI(next.View()); !strings.Contains(out, "Confirmed!") {
		t.Errorf("View() after ConfirmedMsg missing Confirmed!:\n%s", out)
	}
}
