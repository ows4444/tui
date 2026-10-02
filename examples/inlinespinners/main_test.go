package main

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
)

func TestProgressMarksRows(t *testing.T) {
	var m tui.Model = initialModel()
	m, _ = m.Update(doneMsg{0})
	m, _ = m.Update(doneMsg{1})
	rows := strings.Split(m.View(), "\n")
	if len(rows) != len(tasks) {
		t.Fatalf("rows = %d", len(rows))
	}
	if !strings.HasPrefix(rows[0], "[x]") || !strings.HasPrefix(rows[1], "[x]") {
		t.Fatalf("done rows wrong: %q", rows)
	}
	if !strings.HasPrefix(rows[3], "[ ]") || strings.HasPrefix(rows[2], "[") {
		t.Fatalf("pending/running rows wrong: %q", rows)
	}
}

func TestLastTaskQuits(t *testing.T) {
	var m tui.Model = initialModel()
	var cmd tui.Cmd
	for i := range tasks {
		m, cmd = m.Update(doneMsg{i})
	}
	if cmd == nil {
		t.Fatal("expected Quit when all tasks finish")
	}
}
