package tui

import (
	"testing"
	"time"

	"github.com/ows4444/tui/input"
)

func TestSplitKeyAction(t *testing.T) {
	a := Key{Type: KeyRunes, Text: "a", Code: 'a'}
	if got := splitKeyAction(a); got != nil {
		if _, ok := got.(Key); !ok {
			t.Fatalf("press changed: %T", got)
		}
	}
	rel := a
	rel.Action = KeyRelease
	if m, ok := splitKeyAction(rel).(KeyReleaseMsg); !ok || m.Key.Action != KeyRelease {
		t.Fatalf("release -> %T", splitKeyAction(rel))
	}
	rep := a
	rep.Action = KeyRepeat
	if _, ok := splitKeyAction(rep).(KeyRepeatMsg); !ok {
		t.Fatalf("repeat -> %T", splitKeyAction(rep))
	}
}

// A tap (press then release) of "g" must not complete a "g g" chord.
func TestChordIgnoresRelease(t *testing.T) {
	c := input.NewChordMatcher(input.ChordDef{Name: "gg", Keys: []string{"g", "g"}})
	now := time.Now()
	g := input.Key{Type: input.KeyRunes, Text: "g", Code: 'g'}
	if r := c.Feed(g, now); !r.Pending {
		t.Fatalf("press should be pending: %+v", r)
	}
	rel := g
	rel.Action = input.KeyRelease
	r := c.Feed(rel, now)
	if r.Msg != nil || r.Pending {
		t.Fatalf("release completed/extended chord: %+v", r)
	}
	if r := c.Feed(g, now); r.Msg == nil {
		t.Fatalf("second real press should still complete: %+v", r)
	}
}
