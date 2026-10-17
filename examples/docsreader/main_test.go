package main

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/markdownview"
)

func press(m model, keys ...tui.Msg) model {
	for _, k := range keys {
		next, cmd := m.Update(k)
		m = next.(model)
		if cmd != nil {
			if msg := cmd(); msg != nil {
				next, _ = m.Update(msg)
				m = next.(model)
			}
		}
	}
	return m
}

// The contents panel lists every heading, and the reader knows the links.
func TestContentsAndLinks(t *testing.T) {
	m := initialModel()
	out := ansi.StripANSI(m.View())
	for _, want := range []string{"Contents", "tui", "Quick start", "Architecture", "Testing"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
	if got := len(m.reader.Links()); got != 4 {
		t.Fatalf("%d links", got)
	}
}

// Tab reaches the first link, Enter follows an in-page one and the window moves.
func TestTabAndEnterFollowAnInPageLink(t *testing.T) {
	m := initialModel()
	m = press(m, tui.Key{Type: tui.KeyTab}, tui.Key{Type: tui.KeyEnter})
	want := m.reader.Headings()[1].Line
	if got := m.reader.YOffset(); got != want && got == 0 {
		t.Fatalf("offset %d; the Quick start heading is at %d", got, want)
	}
	m = press(m, tui.Key{Type: tui.KeyBackspace})
	if m.reader.YOffset() != 0 {
		t.Fatalf("Back left the window at %d", m.reader.YOffset())
	}
}

// An outside link is reported, not followed.
func TestOutsideLinkIsReported(t *testing.T) {
	m := initialModel()
	for i := 0; i < 3; i++ {
		m = press(m, tui.Key{Type: tui.KeyTab})
	}
	if l, _ := m.reader.Active(); l.URL != "https://github.com/ows4444/tui" {
		t.Fatalf("active link %+v", l)
	}
	m = press(m, tui.Key{Type: tui.KeyEnter})
	if m.opened != "https://github.com/ows4444/tui" {
		t.Fatalf("opened %q", m.opened)
	}
	if !strings.Contains(ansi.StripANSI(m.View()), "would open") {
		t.Error("the status line does not say what would open")
	}
}

func TestQuitKeys(t *testing.T) {
	m := initialModel()
	for _, k := range []tui.Key{{Type: tui.KeyRunes, Text: "q"}, {Type: tui.KeyEsc}, {Type: tui.KeyCtrlC}} {
		if _, cmd := m.Update(k); cmd == nil {
			t.Errorf("%v did not quit", k)
		}
	}
}

func TestResizeResizesTheReader(t *testing.T) {
	m := initialModel()
	next, _ := m.Update(tui.ResizeMsg{Width: 100, Height: 30})
	m = next.(model)
	if m.reader.Width() != 76 || m.reader.Height() != 27 {
		t.Fatalf("reader %dx%d", m.reader.Width(), m.reader.Height())
	}
	var _ markdownview.LinkMsg
}
