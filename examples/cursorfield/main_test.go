package main

import (
	"testing"

	"github.com/ows4444/tui"
)

var _ tui.CursorPlacer = model{}

func TestCursorFollowsTypedText(t *testing.T) {
	var m tui.Model = initialModel()
	x0, y, ok := m.(tui.CursorPlacer).CursorPos()
	if !ok || y != 1 {
		t.Fatalf("CursorPos = %d,%d,%v, want row 1 and ok", x0, y, ok)
	}
	for _, r := range "abc" {
		m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: string([]rune{r})})
	}
	x, y, ok := m.(tui.CursorPlacer).CursorPos()
	if !ok || y != 1 || x != x0+3 {
		t.Fatalf("after typing CursorPos = %d,%d,%v, want %d,1,true", x, y, ok, x0+3)
	}
}

func TestEnterQuits(t *testing.T) {
	_, cmd := initialModel().Update(tui.Key{Type: tui.KeyEnter})
	if cmd == nil {
		t.Fatal("no Cmd")
	}
	if _, ok := cmd().(tui.QuitMsg); !ok {
		t.Fatal("Enter did not quit")
	}
}
