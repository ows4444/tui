package tui

import (
	"testing"
	"time"
)

// describe turns recorded Msgs into short strings so a sequence reads as
// data: keys as their String(), chords as "chord:<name>".
func describe(msgs []Msg) []string {
	var out []string
	for _, m := range msgs {
		switch v := m.(type) {
		case Key:
			out = append(out, v.String())
		case ChordMsg:
			out = append(out, "chord:"+v.Name)
		}
	}
	return out
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

var gg = ChordDef{Name: "top", Keys: []string{"g", "g"}}
var save = ChordDef{Name: "save", Keys: []string{"ctrl+x", "ctrl+s"}}

func runKeys(t *testing.T, opts []ProgramOption, quitAfter int, sends ...string) []string {
	t.Helper()
	write, _, wait := bgRun(t, recorderModel{quitAfter: quitAfter}, opts...)
	for _, s := range sends {
		write(s)
		time.Sleep(15 * time.Millisecond) // separate reads, as typed keys are
	}
	rec := wait().model.(recorderModel)
	return describe(rec.received[1:]) // drop the seeded ResizeMsg
}

func TestChordsOffDeliversKeysUnchanged(t *testing.T) {
	got := runKeys(t, nil, 3, "g", "g")
	if !equal(got, []string{"g", "g"}) {
		t.Errorf("without WithChords keys were altered: %v", got)
	}
}

func TestChordCompletionReplacesItsKeys(t *testing.T) {
	got := runKeys(t, []ProgramOption{WithChords(gg)}, 2, "g", "g")
	if !equal(got, []string{"chord:top"}) {
		t.Errorf("got %v, want a single chord:top", got)
	}
	got = runKeys(t, []ProgramOption{WithChords(save)}, 2, "\x18", "\x13") // ctrl+x, ctrl+s
	if !equal(got, []string{"chord:save"}) {
		t.Errorf("ctrl chord: got %v", got)
	}
}

func TestChordDeadEndReplaysKeysInOrder(t *testing.T) {
	// "g" starts a chord, "x" breaks it: both are delivered, in order.
	got := runKeys(t, []ProgramOption{WithChords(gg)}, 3, "g", "x")
	if !equal(got, []string{"g", "x"}) {
		t.Errorf("got %v, want [g x]", got)
	}
}

func TestPendingPrefixFlushesOnTimeoutWithoutAnotherKey(t *testing.T) {
	// One "g" and then nothing: after the timeout it must still arrive.
	start := time.Now()
	got := runKeys(t, []ProgramOption{WithChords(gg), WithChordTimeout(60 * time.Millisecond)}, 2, "g")
	if !equal(got, []string{"g"}) {
		t.Errorf("got %v, want [g]", got)
	}
	if e := time.Since(start); e < 55*time.Millisecond {
		t.Errorf("delivered after %v, before the 60ms timeout", e)
	}
}

func TestNonChordKeyIsDeliveredImmediately(t *testing.T) {
	// A long timeout: if "x" were held back this would hang until it passed.
	start := time.Now()
	got := runKeys(t, []ProgramOption{WithChords(gg), WithChordTimeout(time.Hour)}, 2, "x")
	if !equal(got, []string{"x"}) {
		t.Errorf("got %v, want [x]", got)
	}
	if e := time.Since(start); e > time.Second {
		t.Errorf("non-chord key took %v", e)
	}
}

func TestChordsAndOrdinaryKeysMix(t *testing.T) {
	got := runKeys(t, []ProgramOption{WithChords(gg)}, 4, "a", "g", "g", "b")
	if !equal(got, []string{"a", "chord:top", "b"}) {
		t.Errorf("got %v", got)
	}
}
