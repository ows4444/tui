package form

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/input"
	"github.com/ows4444/tui/layout"
)

func richForm() Model {
	m := New(
		Field{Name: "title", Label: "Title", Description: "A short name", Validators: []Validator{Required()}},
		Field{Name: "tags", Label: "Tags", Kind: FieldMultiSelect, Options: []string{"go", "tui", "docs"}, Value: "tui"},
		Field{Name: "notes", Label: "Notes", Kind: FieldTextArea, Height: 3, Description: "Anything else"},
	)
	m.Focus()
	return m
}

func TestMultiSelectFieldValueAndSelected(t *testing.T) {
	m := richForm()
	if got := m.Values()["tags"]; got != "tui" {
		t.Fatalf("initial value = %q", got)
	}
	m = tab(m) // to tags
	if m.Current() != "tags" {
		t.Fatalf("current %q", m.Current())
	}
	m, _ = m.Update(tui.Key{Type: tui.KeySpace}) // ticks "go" (cursor on the first)
	m, _ = m.Update(tui.Key{Type: tui.KeyDown})
	m, _ = m.Update(tui.Key{Type: tui.KeyDown})
	m, _ = m.Update(tui.Key{Type: tui.KeySpace}) // ticks "docs"
	if got := m.Values()["tags"]; got != "go,tui,docs" {
		t.Errorf("value = %q, want go,tui,docs", got)
	}
	if got := Selected(m, "tags"); strings.Join(got, "|") != "go|tui|docs" {
		t.Errorf("Selected = %v", got)
	}
	if Selected(m, "title") != nil || Selected(m, "nope") != nil {
		t.Error("Selected of a non-multi field must be nil")
	}
}

func TestTextAreaTakesEnterAndCtrlSSubmits(t *testing.T) {
	m := richForm()
	m = typed(m, "Spec")
	m = tab(tab(m)) // to notes
	if m.Current() != "notes" {
		t.Fatalf("current %q", m.Current())
	}
	m = typed(m, "line one")
	m, cmd := m.Update(tui.Key{Type: tui.KeyEnter})
	if cmd != nil {
		if _, ok := cmd().(SubmittedMsg); ok {
			t.Fatal("Enter in a text area submitted the form")
		}
	}
	m = typed(m, "line two")
	if got := m.Values()["notes"]; got != "line one\nline two" {
		t.Fatalf("notes = %q", got)
	}
	_, cmd = m.Update(tui.Key{Type: tui.KeyRunes, Text: "s", Code: 's', Mod: input.ModCtrl})
	if cmd == nil {
		t.Fatal("Ctrl+S in a text area returned no Cmd")
	}
	sub, ok := cmd().(SubmittedMsg)
	if !ok || sub.Values["notes"] != "line one\nline two" || sub.Values["title"] != "Spec" {
		t.Errorf("submitted = %#v", cmd())
	}
}

func TestDescriptionIsDrawnAndSpoken(t *testing.T) {
	m := richForm()
	v := plain(m)
	if !strings.Contains(v, "A short name") || !strings.Contains(v, "Anything else") {
		t.Errorf("descriptions missing:\n%s", v)
	}
	if got := m.Linearize(); !strings.Contains(got, "A short name") || !strings.Contains(got, "multi-select") || !strings.Contains(got, "text area") {
		t.Errorf("Linearize:\n%s", got)
	}
}

func TestMultiRowFieldsSizeTheLayoutNode(t *testing.T) {
	m := richForm()
	sz := m.LayoutNode().Measure(layout.Unconstrained())
	// title 1 + desc 1; tags label 1 + 3 options; notes label 1 + 3 rows + desc 1
	if sz.H != 2+4+5 {
		t.Errorf("Measure H = %d, want 11", sz.H)
	}
	out := strings.Split(ansi.StripANSI(m.LayoutNode().Render(layout.Size{W: 40, H: 11})), "\n")
	if len(out) != 11 || !strings.HasPrefix(out[2], "Tags:") {
		t.Errorf("Render:\n%s", strings.Join(out, "\n"))
	}
}

func TestMouseRowsSkipMultiRowFields(t *testing.T) {
	m := richForm()
	m.Mouse = true
	m.Bounds = hittest.Rect{X: 0, Y: 0, W: 40, H: 12}
	// Rows: 0 title, 1 its description, 2 "Tags:", 3-5 options, 6 "Notes:", 7-9 area, 10 description.
	m, _ = m.Update(tui.MouseEvent{X: 3, Y: 8, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	if m.Current() != "notes" {
		t.Errorf("click on a text-area row focused %q, want notes", m.Current())
	}
	m, _ = m.Update(tui.MouseEvent{X: 3, Y: 4, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	if m.Current() != "tags" {
		t.Errorf("click on a multi-select row focused %q, want tags", m.Current())
	}
}

func TestBindingsOfTheNewKinds(t *testing.T) {
	m := richForm()
	m = tab(m)
	if b := m.Bindings(); len(b) < 4 {
		t.Errorf("multi-select bindings = %d", len(b))
	}
	m = tab(m)
	var found bool
	for _, b := range m.Bindings() {
		if b.Desc == "submit" && len(b.Keys) == 1 && b.Keys[0] == "ctrl+s" {
			found = true
		}
	}
	if !found {
		t.Error("the text area's Ctrl+S submit binding is not listed")
	}
}
