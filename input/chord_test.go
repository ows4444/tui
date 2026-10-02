package input

import (
	"reflect"
	"testing"
	"time"
)

func rn(r rune) Key { return Key{Type: KeyRunes, Text: string([]rune{r})} }

func TestChordMatcher_CompletesWithinTimeout(t *testing.T) {
	m := NewChordMatcher(ChordDef{Name: "delete-line", Keys: []string{"d", "d"}})
	now := time.Now()

	res := m.Feed(rn('d'), now)
	if !res.Pending || res.Msg != nil || res.Flush != nil {
		t.Fatalf("first 'd' = %+v, want Pending with no Msg/Flush", res)
	}

	res = m.Feed(rn('d'), now.Add(100*time.Millisecond))
	if res.Msg == nil || res.Msg.Name != "delete-line" {
		t.Fatalf("second 'd' = %+v, want ChordMsg{Name: %q}", res, "delete-line")
	}
	if len(res.Flush) != 0 {
		t.Errorf("completed chord should not flush anything, got %v", res.Flush)
	}
}

func TestChordMatcher_TimeoutFlushesBufferedKeys(t *testing.T) {
	m := NewChordMatcher(ChordDef{Name: "delete-line", Keys: []string{"d", "d"}})
	m.Timeout = 200 * time.Millisecond
	now := time.Now()

	res := m.Feed(rn('d'), now)
	if !res.Pending {
		t.Fatalf("first 'd' = %+v, want Pending", res)
	}

	// A later, unrelated key arrives after the chord window has elapsed:
	// the stale 'd' must be delivered, not silently dropped.
	res = m.Feed(rn('x'), now.Add(300*time.Millisecond))
	want := []Key{rn('d'), rn('x')}
	if !reflect.DeepEqual(res.Flush, want) {
		t.Errorf("Flush = %v, want %v (expired prefix, then the new key)", res.Flush, want)
	}
	if res.Msg != nil || res.Pending {
		t.Errorf("expired-then-unrelated key should not match or stay pending: %+v", res)
	}
}

func TestChordMatcher_ExpireWithoutFurtherInput(t *testing.T) {
	m := NewChordMatcher(ChordDef{Name: "delete-line", Keys: []string{"d", "d"}})
	m.Timeout = 100 * time.Millisecond
	now := time.Now()

	m.Feed(rn('d'), now)
	if flushed := m.Expire(now.Add(50 * time.Millisecond)); flushed != nil {
		t.Errorf("Expire before timeout = %v, want nil", flushed)
	}
	flushed := m.Expire(now.Add(200 * time.Millisecond))
	if !reflect.DeepEqual(flushed, []Key{rn('d')}) {
		t.Errorf("Expire after timeout = %v, want [d]", flushed)
	}
	// Buffer is cleared: a fresh 'd' starts a new pending chord, not a
	// stale one already reported by Expire.
	res := m.Feed(rn('d'), now.Add(210*time.Millisecond))
	if !res.Pending {
		t.Errorf("Feed after Expire = %+v, want a fresh Pending", res)
	}
}

func TestChordMatcher_NonChordKeyPassesThroughImmediately(t *testing.T) {
	m := NewChordMatcher(ChordDef{Name: "delete-line", Keys: []string{"d", "d"}})
	res := m.Feed(rn('x'), time.Now())
	if !reflect.DeepEqual(res.Flush, []Key{rn('x')}) || res.Pending || res.Msg != nil {
		t.Errorf("unrelated key = %+v, want immediate Flush of just that key", res)
	}
}

func TestChordMatcher_DeadEndPrefixRetriesKeyAsFreshStart(t *testing.T) {
	m := NewChordMatcher(
		ChordDef{Name: "delete-line", Keys: []string{"d", "d"}},
		ChordDef{Name: "go-top", Keys: []string{"g", "g"}},
	)
	now := time.Now()

	res := m.Feed(rn('d'), now)
	if !res.Pending {
		t.Fatalf("'d' = %+v, want Pending", res)
	}
	// "d g" matches nothing, but the buffered 'd' shouldn't swallow a 'g'
	// that could start its own chord ("g g").
	res = m.Feed(rn('g'), now.Add(10*time.Millisecond))
	if !res.Pending {
		t.Fatalf("'d' then 'g' = %+v, want Pending (started 'g g')", res)
	}
	if !reflect.DeepEqual(res.Flush, []Key{rn('d')}) {
		t.Errorf("Flush = %v, want the dead-end 'd' flushed", res.Flush)
	}

	res = m.Feed(rn('g'), now.Add(20*time.Millisecond))
	if res.Msg == nil || res.Msg.Name != "go-top" {
		t.Errorf("final 'g' = %+v, want ChordMsg{Name: go-top}", res)
	}
}
