package main

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// View combines Gauge, CodeBlock, DiffView, markdown.Render, a
// streamtext/Typewriter model, TokenCounter and layout.JoinVertical.
func TestViewCombinesAllWidgets(t *testing.T) {
	m := initialModel()
	out := ansi.StripANSI(m.View())

	for _, want := range []string{
		"Chat",            // widgets.HeaderWithAccessory
		"parseArgs",       // markdown.Render'd user prompt
		"func parseArgs",  // widgets.CodeBlock
		"+func parseArgs", // widgets.DiffView
		"tokens",          // widgets.TokenCounter
	} {
		if !strings.Contains(out, want) {
			t.Errorf("View() missing %q\n%s", want, out)
		}
	}
}

// Every rendered line, including the padding layout.JoinVertical adds,
// stays within boxWidth.
func TestViewWidthStaysWithinBox(t *testing.T) {
	m := initialModel()
	for _, line := range strings.Split(m.View(), "\n") {
		if w := ansi.Width(line); w > boxWidth {
			t.Errorf("line width %d exceeds boxWidth %d: %q", w, boxWidth, line)
		}
	}
}

func TestThemeToggle(t *testing.T) {
	m := initialModel()
	if m.light {
		t.Fatal("model should start on the dark theme")
	}
	next, _ := m.Update(tui.Key{Type: tui.KeyRunes, Text: "t"})
	m2 := next.(model)
	if !m2.light {
		t.Fatal("'t' should switch to the light theme")
	}
	if m2.reply.Theme != theme.LightTheme() {
		t.Fatal("'t' should update the reply model's theme too")
	}
}

func TestQuitKeys(t *testing.T) {
	m := initialModel()
	for _, key := range []tui.Key{
		{Type: tui.KeyCtrlC},
		{Type: tui.KeyEsc},
		{Type: tui.KeyRunes, Text: "q"},
	} {
		_, cmd := m.Update(key)
		if cmd == nil {
			t.Errorf("key %+v should return a quit Cmd", key)
		}
	}
}
