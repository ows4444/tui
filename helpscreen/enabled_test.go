package helpscreen

import (
	"testing"

	"github.com/ows4444/tui/keymap"
)

func TestFromRegistryOmitsDisabledBindings(t *testing.T) {
	var r keymap.Registry
	r.Add(keymap.Binding{Keys: []string{"q"}, Desc: "quit", Scope: "s"})
	r.Add(keymap.Binding{Keys: []string{"w"}, Desc: "write", Scope: "s"})
	r.Add(keymap.Binding{Keys: []string{"x"}, Desc: "other", Scope: "t"})
	r.SetEnabled("s", "q", false)
	m := FromRegistry(&r, "s")
	if len(m.Hints) != 1 || m.Hints[0].Action != "write" {
		t.Fatalf("Hints = %+v", m.Hints)
	}
	if h := FromKeymap(&r, "s").Hints; len(h) != 1 || h[0].Action != "write" {
		t.Fatalf("FromKeymap Hints = %+v", h)
	}
}
