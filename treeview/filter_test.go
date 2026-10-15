package treeview

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/layout"
)

func typeText(m Model, s string) Model {
	for _, r := range s {
		m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: string(r), Code: r})
	}
	return m
}

func filterModel() Model {
	m := New(
		Node{Label: "src", Children: []Node{
			{Label: "main.go"},
			{Label: "widget", Children: []Node{{Label: "button.go"}, {Label: "slider.go"}}},
		}},
		Node{Label: "docs", Children: []Node{{Label: "readme.md"}}},
		Node{Label: "Makefile"},
	)
	m.Filter = true
	return m
}

func TestFilterOffIgnoresTyping(t *testing.T) {
	m := typeText(New(Node{Label: "a"}, Node{Label: "b"}), "b")
	if m.Query() != "" || m.Cursor() != 0 {
		t.Errorf("Filter off: query %q cursor %d", m.Query(), m.Cursor())
	}
}

func TestFilterShowsMatchWithAncestorsOpened(t *testing.T) {
	m := typeText(filterModel(), "slider")
	rows := m.VisibleRows()
	var labels []string
	for _, r := range rows {
		labels = append(labels, r.Label)
	}
	want := []string{"src", "widget", "slider.go"}
	if strings.Join(labels, ",") != strings.Join(want, ",") {
		t.Fatalf("rows = %v, want %v", labels, want)
	}
	if got := m.flatten()[m.Cursor()].label; got != "slider.go" {
		t.Errorf("cursor on %q, want slider.go", got)
	}
	view := ansi.StripANSI(m.View())
	if !strings.HasPrefix(view, "/ slider") || strings.Contains(view, "docs") {
		t.Errorf("view:\n%s", view)
	}
}

func TestFilterKeepsTreeOrder(t *testing.T) {
	m := typeText(filterModel(), "o")
	var labels []string
	for _, r := range m.VisibleRows() {
		labels = append(labels, r.Label)
	}
	got := strings.Join(labels, ",")
	if !strings.HasPrefix(got, "src,") || strings.Index(got, "docs") < strings.Index(got, "src") {
		t.Errorf("order = %s", got)
	}
}

func TestFilterSelectLeafGivesOriginalPath(t *testing.T) {
	m := typeText(filterModel(), "button")
	_, cmd := m.Update(tui.Key{Type: tui.KeyEnter})
	if cmd == nil {
		t.Fatal("Enter on a leaf returned no Cmd")
	}
	msg := cmd().(SelectedMsg)
	if msg.Path != "0.1.0" || msg.Node.Label != "button.go" {
		t.Errorf("SelectedMsg = %+v", msg)
	}
}

func TestFilterQueryOpensBranchesWithoutChangingExpandState(t *testing.T) {
	m := typeText(filterModel(), "readme")
	if m.IsExpanded("1") {
		t.Error("the query must not write expand state")
	}
	m, _ = m.Update(tui.Key{Type: tui.KeyEsc})
	if m.Query() != "" || len(m.VisibleRows()) != 3 {
		t.Errorf("after Esc: %q, %d rows (want the 3 collapsed roots)", m.Query(), len(m.VisibleRows()))
	}
}

func TestFilterNoMatchesThenBackspace(t *testing.T) {
	m := typeText(filterModel(), "zzz")
	if got := ansi.StripANSI(m.View()); !strings.Contains(got, "no matches") {
		t.Errorf("view:\n%s", got)
	}
	m = typeText(m, "") // no-op
	for range 3 {
		m, _ = m.Update(tui.Key{Type: tui.KeyBackspace})
	}
	if m.Query() != "" || len(m.VisibleRows()) != 3 {
		t.Errorf("after Backspace: %q, %d rows", m.Query(), len(m.VisibleRows()))
	}
}

func TestFilterSelectOnForcedBranchDoesNotToggle(t *testing.T) {
	m := typeText(filterModel(), "slider")
	m.cursor = 0 // "src", open only because of the query
	m, _ = m.Update(tui.Key{Type: tui.KeyEnter})
	if m.IsExpanded("0") {
		t.Error("Enter on a query-opened branch changed expand state")
	}
}

func TestFilterMouseClick(t *testing.T) {
	m := typeText(filterModel(), "slider")
	m.Mouse = true
	m.Bounds = hittest.Rect{X: 0, Y: 0, W: 30, H: 6}
	m, _ = m.Update(tui.MouseEvent{X: 2, Y: 2, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	if m.Cursor() != 1 { // row 0 is the query line; y=2 is the second row, "widget"
		t.Errorf("Cursor = %d, want 1", m.Cursor())
	}
}

func TestFilterLinearize(t *testing.T) {
	m := typeText(filterModel(), "slider")
	want := strings.Join([]string{
		`Filter "slider", 3 rows`,
		"src, level 1, expanded, item 1 of 1",
		"widget, level 2, expanded, item 1 of 1",
		"slider.go, level 3, item 1 of 1, selected",
	}, "\n")
	if got := m.Linearize(); got != want {
		t.Errorf("Linearize =\n%s\nwant\n%s", got, want)
	}
	if got := typeText(m, "zz").Linearize(); got != `Filter "sliderzz", no matches` {
		t.Errorf("no match = %q", got)
	}
}

func TestFilterLayoutNodeKeepsQueryLine(t *testing.T) {
	m := typeText(filterModel(), "o")
	lines := strings.Split(ansi.StripANSI(m.LayoutNode().Render(layout.Size{W: 30, H: 3})), "\n")
	if len(lines) != 3 || !strings.HasPrefix(lines[0], "/ o") {
		t.Errorf("lines = %q", lines)
	}
	if sz := typeText(filterModel(), "zzz").LayoutNode().Measure(layout.Constraints{MaxW: 40, MaxH: 40}); sz.H != 2 {
		t.Errorf("no-match Measure H = %d, want 2", sz.H)
	}
}
