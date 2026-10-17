package typeahead

import (
	"testing"
	"time"

	"github.com/ows4444/tui/input"
)

var fruit = []string{"Apple", "Avocado", "Banana", "Blueberry", "blackberry", "Cherry"}

func label(i int) string { return fruit[i] }

func TestFindGoesRoundAndIgnoresCase(t *testing.T) {
	for _, tc := range []struct {
		prefix string
		from   int
		want   int
		ok     bool
	}{
		{"b", 1, 2, true},
		{"b", 3, 3, true}, // from itself counts
		{"b", 5, 2, true}, // round past the end
		{"B", 0, 2, true},
		{"bl", 3, 3, true},
		{"x", 0, 0, false},
		{"", 0, 0, false},
		{"a", 6, 0, true},
		{"cherry", 0, 5, true},
		{"cherryx", 0, 0, false},
		{"b", -3, 3, true}, // a stray start is wrapped, not a panic
		{"b", 99, 3, true},
	} {
		if i, ok := Find(tc.prefix, tc.from, len(fruit), label); ok != tc.ok || (ok && i != tc.want) {
			t.Errorf("Find(%q, from %d) = %d, %v; want %d, %v", tc.prefix, tc.from, i, ok, tc.want, tc.ok)
		}
	}
	if _, ok := Find("a", 0, 0, label); ok {
		t.Error("an empty list matched")
	}
	calls := 0
	Find("a", 0, 1_000_000, func(i int) string { calls++; return "a" })
	if calls != 1 {
		t.Errorf("Find called label %d times for a first-item match", calls)
	}
}

func TestFeedBuildsAWordAndResetsAfterAPause(t *testing.T) {
	clock := time.Unix(1000, 0)
	Now = func() time.Time { return clock }
	defer func() { Now = time.Now }()
	var s State
	if p, from := s.Feed("b", 4); p != "b" || from != 5 {
		t.Errorf("first letter: %q from %d, want b from 5 (after the cursor)", p, from)
	}
	if p, from := s.Feed("b", 5); p != "b" || from != 6 {
		t.Errorf("same letter again: %q from %d, want b from 6 (it cycles)", p, from)
	}
	if p, from := s.Feed("l", 6); p != "bbl" || from != 4 {
		t.Errorf("mixed word: %q from %d, want bbl from 4 (where the word began)", p, from)
	}
	clock = clock.Add(2 * time.Second)
	if p, from := s.Feed("l", 9); p != "l" || from != 10 {
		t.Errorf("after a pause: %q from %d, want l from 10", p, from)
	}
	s.Clear()
	if p, from := s.Feed("é", 0); p != "é" || from != 1 {
		t.Errorf("a non-ASCII letter: %q from %d", p, from)
	}
	if p, from := s.Feed("é", 1); p != "é" || from != 2 {
		t.Errorf("a repeated non-ASCII letter: %q from %d", p, from)
	}
}

func TestPrintable(t *testing.T) {
	for name, tc := range map[string]struct {
		msg  any
		text string
		ok   bool
	}{
		"letter":     {input.Key{Type: input.KeyRunes, Text: "a"}, "a", true},
		"shifted":    {input.Key{Type: input.KeyRunes, Text: "A", Mod: input.ModShift}, "A", true},
		"ctrl":       {input.Key{Type: input.KeyRunes, Text: "a", Mod: input.ModCtrl}, "", false},
		"alt":        {input.Key{Type: input.KeyRunes, Text: "a", Mod: input.ModAlt}, "", false},
		"release":    {input.Key{Type: input.KeyRunes, Text: "a", Action: input.KeyRelease}, "", false},
		"enter":      {input.Key{Type: input.KeyEnter}, "", false},
		"empty":      {input.Key{Type: input.KeyRunes}, "", false},
		"not a key":  {"a", "", false},
		"multi-rune": {input.Key{Type: input.KeyRunes, Text: "日本"}, "日本", true},
	} {
		if text, ok := Printable(tc.msg); text != tc.text || ok != tc.ok {
			t.Errorf("%s: Printable = %q, %v", name, text, ok)
		}
	}
}
