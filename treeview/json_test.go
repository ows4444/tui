package treeview_test

import (
	"github.com/ows4444/tui/treeview"
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

func TestObjectBecomesSortedChildren(t *testing.T) {
	m, err := treeview.FromJSON([]byte(`{"name":"Alice","age":30}`))
	if err != nil {
		t.Fatalf("FromJSON() error = %v", err)
	}
	if len(m.Roots) != 1 {
		t.Fatalf("Roots = %d, want 1", len(m.Roots))
	}
	root := m.Roots[0]
	if root.Label != "root {}" {
		t.Errorf("root.Label = %q, want %q", root.Label, "root {}")
	}
	if len(root.Children) != 2 {
		t.Fatalf("root.Children = %d, want 2", len(root.Children))
	}
	// keys sorted: age, name
	if root.Children[0].Label != `age: 30` {
		t.Errorf("Children[0].Label = %q, want %q", root.Children[0].Label, "age: 30")
	}
	if root.Children[1].Label != `name: "Alice"` {
		t.Errorf("Children[1].Label = %q, want %q", root.Children[1].Label, `name: "Alice"`)
	}
}

func TestArrayBecomesIndexedChildren(t *testing.T) {
	m, err := treeview.FromJSON([]byte(`[1,2,3]`))
	if err != nil {
		t.Fatalf("FromJSON() error = %v", err)
	}
	root := m.Roots[0]
	if root.Label != "root []" {
		t.Errorf("root.Label = %q, want %q", root.Label, "root []")
	}
	want := []string{"0: 1", "1: 2", "2: 3"}
	if len(root.Children) != len(want) {
		t.Fatalf("root.Children = %d, want %d", len(root.Children), len(want))
	}
	for i, w := range want {
		if root.Children[i].Label != w {
			t.Errorf("Children[%d].Label = %q, want %q", i, root.Children[i].Label, w)
		}
	}
}

func TestNestedObjectInArray(t *testing.T) {
	m, err := treeview.FromJSON([]byte(`{"items":[{"id":1}]}`))
	if err != nil {
		t.Fatalf("FromJSON() error = %v", err)
	}
	root := m.Roots[0]
	items := root.Children[0]
	if items.Label != "items []" {
		t.Fatalf("items.Label = %q, want %q", items.Label, "items []")
	}
	if len(items.Children) != 1 {
		t.Fatalf("items.Children = %d, want 1", len(items.Children))
	}
	obj := items.Children[0]
	if obj.Label != "0 {}" {
		t.Fatalf("obj.Label = %q, want %q", obj.Label, "0 {}")
	}
	if len(obj.Children) != 1 || obj.Children[0].Label != "id: 1" {
		t.Fatalf("obj.Children = %+v, want [id: 1]", obj.Children)
	}
}

func TestNullAndBoolLeaves(t *testing.T) {
	m, err := treeview.FromJSON([]byte(`{"a":null,"b":true,"c":false}`))
	if err != nil {
		t.Fatalf("FromJSON() error = %v", err)
	}
	root := m.Roots[0]
	want := []string{"a: null", "b: true", "c: false"}
	for i, w := range want {
		if root.Children[i].Label != w {
			t.Errorf("Children[%d].Label = %q, want %q", i, root.Children[i].Label, w)
		}
	}
}

func TestInvalidJSONReturnsError(t *testing.T) {
	if _, err := treeview.FromJSON([]byte(`{not json`)); err == nil {
		t.Fatal("FromJSON() with invalid JSON should return an error")
	}
}

func TestViewStartsCollapsed(t *testing.T) {
	m, err := treeview.FromJSON([]byte(`{"a":1,"b":2}`))
	if err != nil {
		t.Fatalf("FromJSON() error = %v", err)
	}
	got := m.View()
	cursorStyle := theme.DarkTheme().ResolvedStates().Selected.Bold()
	want := "> ▸ " + cursorStyle.Render("root {}")
	if got != want {
		t.Errorf("View() = %q, want %q", got, want)
	}
}

// #30: an integer above 2^53 is shown as written, not rounded via float64.
func TestLargeIntegerKeepsItsDigits(t *testing.T) {
	m, err := treeview.FromJSON([]byte(`{"id": 9007199254740993}`))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := m.Roots[0].Children[0].Label, "id: 9007199254740993"; got != want {
		t.Fatalf("label = %q, want %q", got, want)
	}
	// And it is what View draws once the root is expanded.
	m, _ = m.Update(tui.Key{Type: tui.KeyRight})
	if out := ansi.StripANSI(m.View()); !strings.Contains(out, "id: 9007199254740993") {
		t.Fatalf("View:\n%s", out)
	}
}

func TestNumbersAreShownAsWritten(t *testing.T) {
	m, err := treeview.FromJSON([]byte(`[1, -0.5, 1e3, 12345678901234567890, 1.10]`))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"0: 1", "1: -0.5", "2: 1e3", "3: 12345678901234567890", "4: 1.10"}
	for i, w := range want {
		if got := m.Roots[0].Children[i].Label; got != w {
			t.Errorf("Children[%d] = %q, want %q", i, got, w)
		}
	}
}

func TestInvalidOrTrailingJSONIsStillAnError(t *testing.T) {
	for _, in := range []string{``, `{`, `{"a":1} x`, `1 2`, `nul`} {
		if _, err := treeview.FromJSON([]byte(in)); err == nil {
			t.Errorf("FromJSON(%q) succeeded, want an error", in)
		}
	}
	if _, err := treeview.FromJSON([]byte(" {\"a\": 1} \n")); err != nil {
		t.Errorf("surrounding whitespace rejected: %v", err)
	}
}
