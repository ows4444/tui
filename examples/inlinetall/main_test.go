package main

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
)

func TestEnterAppendsLine(t *testing.T) {
	var m tui.Model = initialModel()
	for i := 0; i < 3; i++ {
		var cmd tui.Cmd
		m, cmd = m.Update(tui.Key{Type: tui.KeyEnter})
		if cmd != nil {
			t.Fatal("Enter returned a Cmd")
		}
	}
	rows := strings.Split(m.View(), "\n")
	if len(rows) != 4 || rows[3] != "line 3" {
		t.Fatalf("rows = %q, want the header plus lines 1..3", rows)
	}
}

func TestUpdateDoesNotAliasEarlierModel(t *testing.T) {
	m0 := initialModel()
	a, _ := m0.Update(tui.Key{Type: tui.KeyEnter})
	b, _ := m0.Update(tui.Key{Type: tui.KeyEnter})
	if a.View() != b.View() || len(m0.lines) != 1 {
		t.Fatalf("a=%q b=%q m0=%q", a.View(), b.View(), m0.lines)
	}
}

func TestQuitKeys(t *testing.T) {
	for _, k := range []tui.KeyType{tui.KeyEsc, tui.KeyCtrlC} {
		_, cmd := initialModel().Update(tui.Key{Type: k})
		if cmd == nil {
			t.Fatalf("key %v: no Cmd", k)
		}
		if _, ok := cmd().(tui.QuitMsg); !ok {
			t.Fatalf("key %v did not quit", k)
		}
	}
}
