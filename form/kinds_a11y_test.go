package form

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
)

// Linearize speaks a select as its label, that it is a select, whether it has
// focus, its chosen option and its position, and a checkbox as its label and
// checked or unchecked.
func TestLinearizeSpeaksSelectAndCheckbox(t *testing.T) {
	m := kindsForm()
	m = key(m, tui.KeyTab) // focus the select
	m = key(m, tui.KeyRight)
	lines := strings.Split(m.Linearize(), "\n")
	if len(lines) != 3 {
		t.Fatalf("%d lines, want one per field: %q", len(lines), m.Linearize())
	}
	for _, want := range []string{"Plan", "select", "focused", "Pro", "2 of 3"} {
		if !strings.Contains(lines[1], want) {
			t.Errorf("select line %q lacks %q", lines[1], want)
		}
	}
	for _, want := range []string{"Accept terms", "checkbox", "unchecked"} {
		if !strings.Contains(lines[2], want) {
			t.Errorf("checkbox line %q lacks %q", lines[2], want)
		}
	}
	if strings.Contains(lines[2], "focused") {
		t.Errorf("the checkbox is not focused but its line says so: %q", lines[2])
	}
	m = key(m, tui.KeyTab)
	m = space(m)
	line := strings.Split(m.Linearize(), "\n")[2]
	if !strings.Contains(line, "focused") || !strings.Contains(line, "checked") || strings.Contains(line, "unchecked") {
		t.Errorf("a ticked, focused checkbox is spoken as %q", line)
	}
}

func TestLinearizeSpeaksErrorsOnNewKinds(t *testing.T) {
	m, _ := kindsForm().Submit()
	line := strings.Split(m.Linearize(), "\n")[2]
	if !strings.Contains(line, "error: you must accept the terms") {
		t.Errorf("checkbox line %q does not speak its error", line)
	}
}

func TestLinearizeSelectWithNoOptions(t *testing.T) {
	m := New(Field{Name: "p", Label: "Plan", Kind: FieldSelect})
	if got := m.Linearize(); !strings.Contains(got, "Plan") || !strings.Contains(got, "no options") {
		t.Errorf("Linearize = %q, want the label and no options", got)
	}
}

// The layout node is exactly the requested size for every kind, in every
// state, at the sizes the other layout tests use.
func TestLayoutNodeExactSizeForNewKinds(t *testing.T) {
	states := map[string]Model{}
	base := kindsForm()
	states["fresh"] = base
	onSelect := key(base, tui.KeyTab)
	states["select focused"] = onSelect
	onBox := space(key(onSelect, tui.KeyTab))
	states["checkbox ticked"] = onBox
	states["errors"], _ = kindsForm().Submit()
	long := New(Field{Name: "p", Label: "A very long label for a select", Kind: FieldSelect, Options: []string{"an option that is rather long", "b"}},
		Field{Name: "c", Label: "An equally long label for a checkbox", Kind: FieldCheckbox})
	long.Focus()
	states["long labels"] = long
	for name, m := range states {
		for _, s := range []layout.Size{{W: 40, H: 10}, {W: 80, H: 24}, {W: 300, H: 80}, {W: 5, H: 3}, {W: 12, H: 1}} {
			lines := strings.Split(m.LayoutNode().Render(s), "\n")
			if len(lines) != s.H {
				t.Fatalf("%s at %v: %d rows, want %d", name, s, len(lines), s.H)
			}
			for i, l := range lines {
				if w := ansi.Width(l); w != s.W {
					t.Errorf("%s at %v: row %d is %d wide, want %d", name, s, i, w, s.W)
				}
			}
		}
		if got := m.LayoutNode().Measure(layout.Unconstrained()); got.H < len(m.fields) {
			t.Errorf("%s measures %v, want at least one row per field (%d)", name, got, len(m.fields))
		}
	}
}

// The window follows a focused select or checkbox when the form is taller than
// the space, like it does for text fields.
func TestLayoutNodeKeepsAFocusedNewKindVisible(t *testing.T) {
	m := kindsForm()
	one := func(m Model) string { return ansi.StripANSI(m.LayoutNode().Render(layout.Size{W: 40, H: 1})) }
	if got := one(m); !strings.Contains(got, "Name") {
		t.Errorf("first field: %q", got)
	}
	m = key(m, tui.KeyTab)
	if got := one(m); !strings.Contains(got, "Plan") {
		t.Errorf("select focused: %q", got)
	}
	m = key(m, tui.KeyTab)
	if got := one(m); !strings.Contains(got, "Accept terms") {
		t.Errorf("checkbox focused: %q", got)
	}
}

// Under the ASCII theme the new kinds draw only 7-bit characters, in View and
// in the layout node.
func TestNewKindsAreSevenBitUnderTheASCIITheme(t *testing.T) {
	m := kindsForm()
	m.Theme = theme.DarkTheme().ASCII()
	for _, state := range []Model{m, key(m, tui.KeyTab), space(key(key(m, tui.KeyTab), tui.KeyTab))} {
		for _, out := range []string{state.View(), state.LayoutNode().Render(layout.Size{W: 40, H: 6}), state.Linearize()} {
			for _, r := range ansi.StripANSI(out) {
				if r >= 0x80 {
					t.Fatalf("draws %q (U+%04X) under the ASCII theme:\n%s", r, r, ansi.StripANSI(out))
				}
			}
		}
	}
}

// The view shows the state: the chosen option, and the box ticked or not.
func TestViewShowsSelectChoiceAndCheckboxState(t *testing.T) {
	m := kindsForm()
	m = key(m, tui.KeyTab)
	m = key(m, tui.KeyRight)
	v := plain(m)
	if !strings.Contains(v, "Plan: < Pro >") {
		t.Errorf("select not drawn as Plan: < Pro >:\n%s", v)
	}
	if !strings.Contains(v, "[ ] Accept terms") {
		t.Errorf("unticked checkbox not drawn:\n%s", v)
	}
	m = space(key(m, tui.KeyTab))
	if !strings.Contains(plain(m), "[x] Accept terms") {
		t.Errorf("ticked checkbox not drawn:\n%s", plain(m))
	}
}
