package keymap

import (
	"testing"

	"github.com/ows4444/tui/input"
)

// TestDisabledBindingNeverMatches proves criterion #92 for Matches.
func TestDisabledBindingNeverMatches(t *testing.T) {
	b := NewBinding("quit", "q")
	key := input.Key{Type: input.KeyRunes, Text: "q", Code: 'q'}
	if !Matches(key, b) || !b.Enabled() {
		t.Fatal("zero-value binding must be enabled and match")
	}
	b.SetEnabled(false)
	if Matches(key, b) || b.Enabled() {
		t.Fatal("disabled binding matched")
	}
	b.SetEnabled(true)
	if !Matches(key, b) {
		t.Fatal("re-enabled binding did not match")
	}
}

// TestRegistrySetEnabledHidesFromHelp proves criterion #92 for listings.
func TestRegistrySetEnabledHidesFromHelp(t *testing.T) {
	var r Registry
	r.Add(Binding{Keys: []string{"q"}, Desc: "quit", Scope: "s"})
	r.Add(Binding{Keys: []string{"w"}, Desc: "write", Scope: "s"})
	if n := r.SetEnabled("s", "q", false); n != 1 {
		t.Fatalf("SetEnabled = %d", n)
	}
	if h := r.Hints("s"); len(h) != 1 || h[0].Desc != "write" {
		t.Fatalf("Hints = %+v", h)
	}
	if b := r.Bindings("s"); len(b) != 1 {
		t.Fatalf("Bindings = %+v", b)
	}
	r.SetEnabled("s", "q", true)
	if len(r.Hints("s")) != 2 {
		t.Fatal("re-enabled binding not listed")
	}
}
