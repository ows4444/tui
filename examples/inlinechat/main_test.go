package main

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
)

func typeText(m tui.Model, s string) tui.Model {
	m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: s})
	return m
}

func TestSendCommitsAndReplies(t *testing.T) {
	var m tui.Model = model{}
	m = typeText(m, "hi")
	if v := m.View(); !strings.Contains(v, "> hi") {
		t.Fatalf("view = %q", v)
	}
	m, cmd := m.Update(tui.Key{Type: tui.KeyEnter})
	if cmd == nil {
		t.Fatal("enter returned no Cmd")
	}
	if !m.(model).waiting || m.(model).input != "" {
		t.Fatalf("state after send: %+v", m)
	}
	m, cmd = m.Update(replyMsg{text: "bot: x"})
	if cmd == nil || m.(model).waiting {
		t.Fatal("reply should commit a line and stop waiting")
	}
}

func TestEmptyEnterIgnored(t *testing.T) {
	_, cmd := model{}.Update(tui.Key{Type: tui.KeyEnter})
	if cmd != nil {
		t.Fatal("empty enter must not send")
	}
}

func TestQuit(t *testing.T) {
	_, cmd := model{}.Update(tui.Key{Type: tui.KeyEsc})
	if cmd == nil {
		t.Fatal("esc should quit")
	}
}
