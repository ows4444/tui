package keymap

import (
	"testing"

	"github.com/ows4444/tui/input"
)

// TestRegisteredBindingListed proves criterion #93: a binding registered with
// keys and a description is returned by Bindings/Hints, the single list that
// helpscreen and the key-hint widget read.
func TestRegisteredBindingListed(t *testing.T) {
	var r Registry
	if _, err := r.Add(Binding{Keys: []string{"q", "ctrl+c"}, Desc: "quit", Scope: "global"}); err != nil {
		t.Fatal(err)
	}
	got := r.Bindings("global")
	if len(got) != 1 || got[0].Desc != "quit" || got[0].Keys[1] != "ctrl+c" {
		t.Fatalf("Bindings = %+v", got)
	}
	h := r.Hints("global")
	if len(h) != 1 || h[0].Key != "q/ctrl+c" || h[0].Desc != "quit" {
		t.Fatalf("Hints = %+v", h)
	}
}

// TestSameKeySameScopeConflicts proves criterion #94: two bindings on one key
// in one scope are reported; the same key in another scope is not.
func TestSameKeySameScopeConflicts(t *testing.T) {
	var r Registry
	r.Add(Binding{Keys: []string{"a", "x"}, Desc: "one", Scope: "s"})
	c, err := r.Add(Binding{Keys: []string{"x"}, Desc: "two", Scope: "s"})
	if err == nil || len(c) != 1 || c[0].Key != "x" || c[0].Existing.Desc != "one" {
		t.Fatalf("conflicts = %+v, err = %v", c, err)
	}
	if got := r.Conflicts(); len(got) != 1 {
		t.Fatalf("Conflicts() = %+v", got)
	}
	r2 := Registry{}
	r2.Add(Binding{Keys: []string{"x"}, Desc: "one", Scope: "s"})
	if _, err := r2.Add(Binding{Keys: []string{"x"}, Desc: "two", Scope: "t"}); err != nil {
		t.Fatalf("different scope reported conflict: %v", err)
	}
}

func TestMatches(t *testing.T) {
	b := NewBinding("down", "down", "j")
	for _, k := range []input.Key{
		{Type: input.KeyDown},
		{Type: input.KeyRunes, Text: "j"},
		{Type: input.KeyDown, Action: input.KeyRepeat},
	} {
		if !Matches(k, b) {
			t.Errorf("Matches(%q) = false, want true", k.String())
		}
	}
	for _, k := range []input.Key{
		{Type: input.KeyUp},
		{Type: input.KeyRunes, Text: "k"},
		{Type: input.KeyDown, Action: input.KeyRelease},
		{Type: input.KeyRunes, Text: "j", Action: input.KeyRelease},
	} {
		if Matches(k, b) {
			t.Errorf("Matches(%q, action %v) = true, want false", k.String(), k.Action)
		}
	}
	if Matches("j", b) || Matches(nil, b) || Matches(input.Key{Type: input.KeyDown}, Binding{}) {
		t.Error("a non-key Msg or an empty binding must not match")
	}
}

// TestPlainNameAgreesWithKeyString keeps appendPlainName in step with
// input.Key.String for every key type, with and without runes.
func TestPlainNameAgreesWithKeyString(t *testing.T) {
	runes := [][]rune{nil, {'a'}, {' '}, {'é'}, {'x', 'y'}}
	for typ := input.KeyType(0); typ < 64; typ++ {
		for _, rs := range runes {
			k := input.Key{Type: typ, Text: string(rs)}
			if got := string(appendPlainName(nil, k)); got != k.String() {
				t.Errorf("type %d runes %q: appendPlainName = %q, String = %q", typ, string(rs), got, k.String())
			}
		}
	}
}

// TestMatchesDoesNotAllocateForPlainKeys: typing a character asks every
// binding of a widget, so the check must not allocate.
func TestMatchesDoesNotAllocateForPlainKeys(t *testing.T) {
	b := NewBinding("left", "left", "ctrl+b")
	for _, k := range []input.Key{
		{Type: input.KeyRunes, Text: "x"},
		{Type: input.KeyLeft},
		{Type: input.KeyCtrl, Code: 'b'},
	} {
		if n := testing.AllocsPerRun(100, func() { Matches(k, b) }); n != 0 {
			t.Errorf("Matches(%q) allocates %v times", k.String(), n)
		}
	}
}
