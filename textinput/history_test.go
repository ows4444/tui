package textinput

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/cellbuf"
)

func withHistory(entries ...string) Model {
	m := New()
	m.History = entries
	m.Focus()
	return m
}

func key(m Model, t tui.KeyType) Model {
	m, _ = m.Update(tui.Key{Type: t})
	return m
}

func typed(m Model, s string) Model {
	for _, r := range s {
		m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: string(r)})
	}
	return m
}

func ctrl(m Model, r rune) Model {
	m, _ = m.Update(tui.Key{Type: tui.KeyCtrl, Code: r})
	return m
}

func TestUpAndDownWalkTheHistoryAndComeBackToTheDraft(t *testing.T) {
	m := typed(withHistory("one", "two", "three"), "dra")
	m = key(m, tui.KeyUp)
	if m.Value() != "three" {
		t.Fatalf("first Up: %q", m.Value())
	}
	m = key(m, tui.KeyUp)
	m = key(m, tui.KeyUp)
	m = key(m, tui.KeyUp) // already the oldest
	if m.Value() != "one" {
		t.Fatalf("Up past the oldest: %q", m.Value())
	}
	m = key(m, tui.KeyDown)
	if m.Value() != "two" {
		t.Fatalf("Down: %q", m.Value())
	}
	m = key(m, tui.KeyDown)
	m = key(m, tui.KeyDown)
	if m.Value() != "dra" {
		t.Fatalf("Down off the newest should restore the draft, got %q", m.Value())
	}
	if m = key(m, tui.KeyDown); m.Value() != "dra" {
		t.Fatalf("Down with nothing newer changed the value: %q", m.Value())
	}
}

func TestEditingARecalledEntryKeepsThePlace(t *testing.T) {
	m := withHistory("alpha", "beta")
	m = key(m, tui.KeyUp)
	m = typed(m, "!")
	if m.Value() != "beta!" {
		t.Fatalf("%q", m.Value())
	}
	m = key(m, tui.KeyUp)
	if m.Value() != "alpha" {
		t.Fatalf("Up after an edit went to %q; want the next older entry", m.Value())
	}
}

func TestSetValueAndResetForgetThePlace(t *testing.T) {
	m := withHistory("a", "b")
	m = key(m, tui.KeyUp)
	m.Reset()
	m = key(m, tui.KeyUp)
	if m.Value() != "b" {
		t.Fatalf("after Reset, Up gave %q; want the newest", m.Value())
	}
}

func TestNoHistoryMeansUpAndDownDoNothing(t *testing.T) {
	m := typed(withHistory(), "keep")
	if m = key(key(m, tui.KeyUp), tui.KeyDown); m.Value() != "keep" {
		t.Fatalf("%q", m.Value())
	}
}

func TestHistorySearchFindsOlderMatches(t *testing.T) {
	m := typed(withHistory("git status", "ls", "git commit", "make", "git push"), "git")
	m = ctrl(m, 'r')
	if m.Value() != "git push" {
		t.Fatalf("first search: %q", m.Value())
	}
	m = ctrl(m, 'r')
	if m.Value() != "git commit" {
		t.Fatalf("second search: %q", m.Value())
	}
	m = ctrl(m, 'r')
	if m.Value() != "git status" {
		t.Fatalf("third search: %q", m.Value())
	}
	m = ctrl(m, 'r') // nothing older: stay
	if m.Value() != "git status" {
		t.Fatalf("past the oldest match: %q", m.Value())
	}
	m = typed(m, "!") // any other key ends the search
	m = ctrl(m, 'r')
	if m.Value() != "git status!" {
		t.Fatalf("a search for text no entry has should leave the value alone: %q", m.Value())
	}
}

func TestSuggestionShowsAndIsAccepted(t *testing.T) {
	m := New()
	m.Focus()
	m.Suggestion = "hello world"
	m = typed(m, "hel")
	if v := ansi.StripANSI(m.View()); !strings.HasSuffix(strings.TrimRight(v, " "), "lo world") {
		t.Fatalf("no ghost text: %q", v)
	}
	if got := m.suggestionRest(); got != "lo world" {
		t.Fatalf("rest = %q", got)
	}
	m = key(m, tui.KeyRight)
	if m.Value() != "hello world" {
		t.Fatalf("Right at the end should accept: %q", m.Value())
	}
	if m.suggestionRest() != "" {
		t.Fatal("a taken suggestion is still offered")
	}
}

func TestSuggestionNeedsAPrefixAndTheEnd(t *testing.T) {
	m := New()
	m.Focus()
	m.Suggestion = "hello"
	m = typed(m, "hex")
	if m.suggestionRest() != "" {
		t.Fatal("a value that is not a prefix got a suggestion")
	}
	m = New()
	m.Focus()
	m.Suggestion = "hello"
	m = typed(m, "hel")
	m = key(m, tui.KeyLeft)
	if m.suggestionRest() != "" {
		t.Fatal("a suggestion shows with the cursor in the middle")
	}
	empty := New()
	empty.Focus()
	empty.Suggestion = "x"
	if empty.suggestionRest() != "" {
		t.Fatal("an empty value got a suggestion")
	}
}

func TestSuggestFromHistoryPicksTheNewestPrefixMatch(t *testing.T) {
	m := withHistory("git commit", "git push", "ls")
	m.SuggestFromHistory = true
	m = typed(m, "git ")
	if got := m.suggestionRest(); got != "push" {
		t.Fatalf("rest = %q; want push (the newest)", got)
	}
	m = ctrl(m, 'e')
	if m.Value() != "git push" {
		t.Fatalf("Ctrl+E should accept: %q", m.Value())
	}
}

func TestSuggestionHelpAndDrawCells(t *testing.T) {
	m := withHistory("one")
	found := 0
	for _, b := range m.Bindings() {
		if b.Desc == "previous entry" || b.Desc == "search history" {
			found++
		}
	}
	if found != 2 {
		t.Fatal("history keys missing from Bindings")
	}
	plain := New()
	for _, b := range plain.Bindings() {
		if b.Desc == "previous entry" {
			t.Fatal("history keys listed with no history")
		}
	}
}

func TestSuggestionDrawsLikeView(t *testing.T) {
	m := New()
	m.Prompt = "> "
	m.Focus()
	m.Suggestion = "hello world"
	m = typed(m, "hel")
	want, err := cellbuf.Parse(m.View())
	if err != nil {
		t.Fatal(err)
	}
	got := cellbuf.New(30, 1)
	m.DrawCells(got, got.Bounds())
	for x := 0; x < 20; x++ {
		if a, b := want.At(x, 0), got.At(x, 0); a.Cluster != b.Cluster {
			t.Fatalf("column %d: View %q, DrawCells %q", x, a.Cluster, b.Cluster)
		}
	}
}
