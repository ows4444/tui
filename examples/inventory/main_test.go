package main

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
)

func press(m model, keys ...tui.Msg) model {
	for _, k := range keys {
		next, _ := m.Update(k)
		m = next.(model)
	}
	return m
}

func typed(s string) []tui.Msg {
	var out []tui.Msg
	for _, r := range s {
		out = append(out, tui.Key{Type: tui.KeyRunes, Text: string(r)})
	}
	return out
}

func TestPricesAreFormatted(t *testing.T) {
	out := ansi.StripANSI(initialModel().View())
	if !strings.Contains(out, "$1.25") {
		t.Errorf("no formatted price:\n%s", out)
	}
}

func TestMarkingCountsInTheDetailPane(t *testing.T) {
	m := press(initialModel(), tui.Key{Type: tui.KeyRunes, Text: " "}, tui.Key{Type: tui.KeyDown}, tui.Key{Type: tui.KeyRunes, Text: " "})
	if got := len(m.table.Marked()); got != 2 {
		t.Fatalf("%d rows marked", got)
	}
	if !strings.Contains(ansi.StripANSI(m.View()), "Marked: 2") {
		t.Error("the detail pane does not count the marks")
	}
}

func TestDetailFollowsTheCursor(t *testing.T) {
	m := press(initialModel(), tui.Key{Type: tui.KeyDown})
	if !strings.Contains(m.detail(), "nut-0001") {
		t.Fatalf("detail:\n%s", m.detail())
	}
}
