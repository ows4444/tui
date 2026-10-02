package main

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
)

func update(m model, msgs ...tui.Msg) model {
	for _, msg := range msgs {
		next, _ := m.Update(msg)
		m = next.(model)
	}
	return m
}

func runes(r rune) tui.Key { return tui.Key{Type: tui.KeyRunes, Text: string([]rune{r})} }

func TestGGChordJumpsToTopAndCapitalGToBottom(t *testing.T) {
	m := update(initialModel(), runes('G'))
	if !m.vp.AtBottom() || m.vp.AtTop() {
		t.Fatalf("G should jump to the bottom (AtBottom=%v AtTop=%v)", m.vp.AtBottom(), m.vp.AtTop())
	}
	m = update(m, tui.ChordMsg{Name: chordTop})
	if !m.vp.AtTop() {
		t.Error("the gg chord should jump back to the top")
	}
}

func TestUnknownChordAndOrdinaryKeysDoNotMove(t *testing.T) {
	m := update(initialModel(), runes('G'))
	m = update(m, tui.ChordMsg{Name: "something-else"})
	if !m.vp.AtBottom() {
		t.Error("an unrecognised chord must not scroll")
	}
}

func TestQuitKeysStillWork(t *testing.T) {
	for _, k := range []tui.Key{{Type: tui.KeyCtrlC}, {Type: tui.KeyEsc}, runes('q')} {
		if _, cmd := initialModel().Update(k); cmd == nil {
			t.Errorf("%v should quit", k)
		}
	}
	// A lone "g" (what a timed-out, non-completed chord delivers) is just a key.
	if _, cmd := initialModel().Update(runes('g')); cmd != nil {
		t.Error("a plain g should do nothing")
	}
}

func TestHelpMentionsTheBindings(t *testing.T) {
	v := ansi.StripANSI(initialModel().View())
	if !strings.Contains(v, "gg top") || !strings.Contains(v, "G bottom") {
		t.Errorf("help line lacks the new bindings:\n%s", v)
	}
}
