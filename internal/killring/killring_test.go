package killring

import "testing"

func TestKillsAreSavedNewestFirst(t *testing.T) {
	var r Ring
	r = r.Kill("one", false).Other().Kill("two", false)
	if r.Len() != 2 {
		t.Fatalf("Len = %d", r.Len())
	}
	if text, _, ok := r.Yank(); !ok || text != "two" {
		t.Errorf("Yank = %q, %v", text, ok)
	}
}

func TestConsecutiveKillsJoinInDocumentOrder(t *testing.T) {
	var r Ring
	r = r.Kill("c", true).Kill("b", true).Kill("a", true) // backward: prepend
	if text, _, _ := r.Yank(); text != "abc" || r.Len() != 1 {
		t.Errorf("backward kills = %q (%d entries), want abc (1)", text, r.Len())
	}
	r = Ring{}.Kill("a", false).Kill("b", false).Kill("c", false) // forward: append
	if text, _, _ := r.Yank(); text != "abc" || r.Len() != 1 {
		t.Errorf("forward kills = %q (%d entries), want abc (1)", text, r.Len())
	}
	r = Ring{}.Kill("two", true).Kill(" three", false) // a backward then a forward kill
	if text, _, _ := r.Yank(); text != "two three" {
		t.Errorf("mixed kills = %q", text)
	}
}

func TestOtherEndsARunAndEmptyKillsSaveNothing(t *testing.T) {
	r := Ring{}.Kill("a", false).Other().Kill("b", false)
	if r.Len() != 2 {
		t.Errorf("a kill after Other joined the last: %d entries", r.Len())
	}
	r = Ring{}.Kill("", false)
	if r.Len() != 0 {
		t.Error("an empty kill was saved")
	}
	r = Ring{}.Kill("a", false).Kill("", false).Kill("b", false)
	if text, _, _ := r.Yank(); text != "ab" || r.Len() != 1 {
		t.Errorf("an empty kill broke the run: %q (%d)", text, r.Len())
	}
	if _, _, ok := (Ring{}).Yank(); ok {
		t.Error("an empty ring yanked")
	}
}

func TestPopCyclesOlderEntriesOnlyAfterAYank(t *testing.T) {
	r := Ring{}.Kill("a", false).Other().Kill("b", false).Other().Kill("c", false)
	if _, _, ok := r.Pop(); ok {
		t.Error("Pop worked without a yank")
	}
	_, r, _ = r.Yank()
	var got []string
	for i := 0; i < 4; i++ {
		var text string
		var ok bool
		text, r, ok = r.Pop()
		if !ok {
			t.Fatal("Pop failed after a yank")
		}
		got = append(got, text)
	}
	if want := "b a c b"; got[0]+" "+got[1]+" "+got[2]+" "+got[3] != want {
		t.Errorf("Pop order = %v, want %q", got, want)
	}
	if _, _, ok := r.Other().Pop(); ok {
		t.Error("Pop worked after another command")
	}
	one := Ring{}.Kill("x", false)
	_, one, _ = one.Yank()
	if _, _, ok := one.Pop(); ok {
		t.Error("Pop worked with one entry")
	}
}

func TestRingKeepsSixteenAndIsAValue(t *testing.T) {
	var r Ring
	for i := 0; i < 40; i++ {
		r = r.Kill(string(rune('a'+i%26)), false).Other()
	}
	if r.Len() != 16 {
		t.Errorf("Len = %d, want 16", r.Len())
	}
	a := Ring{}.Kill("a", false).Other()
	b := a.Kill("b", false)
	if a.Len() != 1 || b.Len() != 2 {
		t.Errorf("a copy changed the original: %d, %d", a.Len(), b.Len())
	}
	if k, y := b.Last(); !k || y {
		t.Errorf("Last = %v, %v", k, y)
	}
	_, c, _ := b.Yank()
	if k, y := c.Last(); k || !y {
		t.Errorf("Last after a yank = %v, %v", k, y)
	}
}
