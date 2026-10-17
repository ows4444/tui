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

func TestOutlineListsTheFunctions(t *testing.T) {
	out := initialModel().outline()
	for _, want := range []string{"main()", "helper() int"} {
		if !strings.Contains(out, want) {
			t.Errorf("outline lacks %q:\n%s", want, out)
		}
	}
}

func TestFindCountsMatches(t *testing.T) {
	m := press(initialModel(), tui.Key{Type: tui.KeyF2})
	if m.found != 2 {
		t.Fatalf("%d matches for return; want 2", m.found)
	}
	if !strings.Contains(ansi.StripANSI(m.View()), "2 matches") {
		t.Error("status does not say how many")
	}
}

func TestHighlightAddsUp(t *testing.T) {
	line := "  for i := 0; i < 3; i++ {"
	if got := ansi.StripANSI(highlight(line).String()); got != line {
		t.Fatalf("highlight changed the text: %q", got)
	}
}
