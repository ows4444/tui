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

func TestFilterNarrowsTheLines(t *testing.T) {
	m := press(initialModel(), typed("ERROR")...)
	if m.matches == 0 || m.matches >= len(m.lines) {
		t.Fatalf("%d of %d lines match ERROR", m.matches, len(m.lines))
	}
	if !strings.Contains(ansi.StripANSI(m.View()), "ERROR") {
		t.Error("no ERROR line on screen")
	}
}

func TestUpRecallsAndEnterKeeps(t *testing.T) {
	m := press(initialModel(), tui.Key{Type: tui.KeyUp})
	if m.filter.Value() != "WARN gateway" {
		t.Fatalf("Up recalled %q", m.filter.Value())
	}
	m = press(initialModel(), append(typed("api"), tui.Key{Type: tui.KeyEnter})...)
	if h := m.filter.History; h[len(h)-1] != "api" {
		t.Fatalf("history %v", h)
	}
}

func TestEscQuits(t *testing.T) {
	if _, cmd := initialModel().Update(tui.Key{Type: tui.KeyEsc}); cmd == nil {
		t.Fatal("Esc did not quit")
	}
}
