package autocomplete

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
)

func typeText(m Model, s string) Model {
	for _, r := range s {
		m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: string(r), Code: r})
	}
	return m
}

func TestFuzzyOffKeepsPrefixFilter(t *testing.T) {
	m := typeText(New("CommandPalette", "Copy"), "cp")
	if m.IsOpen() {
		t.Errorf("prefix filter matched %v", m.filtered())
	}
}

func TestFuzzyListsSubsequenceMatchesBestFirst(t *testing.T) {
	m := New("CommandPalette", "Banana", "Copy Path")
	m.Fuzzy = true
	m = typeText(m, "cp")
	got := strings.Join(m.filtered(), ",")
	if got != "Copy Path,CommandPalette" {
		t.Errorf("filtered = %s", got)
	}
	view := ansi.StripANSI(m.View())
	if !strings.Contains(view, "> Copy Path") || strings.Contains(view, "Banana") {
		t.Errorf("view:\n%s", view)
	}
	if !strings.Contains(m.View(), "\x1b[") {
		t.Error("matched runes carry no styling")
	}
}

func TestFuzzyAcceptTakesWholeSuggestion(t *testing.T) {
	m := New("CommandPalette", "Copy Path")
	m.Fuzzy = true
	m = typeText(m, "cmd")
	m, cmd := m.Update(tui.Key{Type: tui.KeyEnter})
	if got := cmd().(AcceptedMsg).Value; got != "CommandPalette" {
		t.Errorf("accepted %q", got)
	}
	if m.Input.Value() != "CommandPalette" || m.IsOpen() {
		t.Errorf("input %q open %v", m.Input.Value(), m.IsOpen())
	}
}
