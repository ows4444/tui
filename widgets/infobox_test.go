package widgets

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/theme"
)

func TestInfoBoxFlatUsesKeyValue(t *testing.T) {
	tt := theme.DarkTheme()
	rows := []TreeRow{{Key: "Name", Value: "acline"}, {Key: "Status", Value: "running"}}

	got := InfoBox("Info", rows, tt, 0)
	want := Panel("Info", KeyValue([]KV{{Key: "Name", Value: "acline"}, {Key: "Status", Value: "running"}}, tt), tt, 0)
	if got != want {
		t.Errorf("InfoBox(flat) = %q, want %q (Panel+KeyValue composition)", got, want)
	}
}

func TestInfoBoxEmptyRowsUsesPanelEmptyBehavior(t *testing.T) {
	tt := theme.DarkTheme()
	got := InfoBox("Info", nil, tt, 0)
	want := Panel("Info", "", tt, 0)
	if got != want {
		t.Errorf("InfoBox(nil) = %q, want %q (Panel's own empty-content rendering)", got, want)
	}
}

func TestTreeRowIndentationAcrossDepths(t *testing.T) {
	tt := theme.DarkTheme()
	rows := []TreeRow{
		{Key: "root", Children: []TreeRow{
			{Key: "child", Value: "1", Children: []TreeRow{
				{Key: "grandchild", Value: "2"},
			}},
			{Key: "leaf", Value: "3"},
		}},
	}

	content := renderRows(rows, tt)
	lines := strings.Split(content, "\n")

	wantPrefixes := []string{
		"▾ root",
		"  ▾ child: 1",
		"      grandchild: 2",
		"    leaf: 3",
	}
	if len(lines) != len(wantPrefixes) {
		t.Fatalf("renderRows produced %d lines, want %d:\n%s", len(lines), len(wantPrefixes), content)
	}
	for i, want := range wantPrefixes {
		if !strings.HasPrefix(lines[i], want) {
			t.Errorf("line %d = %q, want prefix %q", i, lines[i], want)
		}
	}
}

func TestTreeRowLeafMarkerIsSpace(t *testing.T) {
	tt := theme.DarkTheme()
	rows := []TreeRow{{Key: "leaf", Value: "x"}}
	// A single flat row with no siblings that nest still counts as "not
	// nested" and goes through KeyValue, so force the tree path with a
	// sibling that has children.
	rows = append(rows, TreeRow{Key: "branch", Children: []TreeRow{{Key: "c"}}})

	content := renderRows(rows, tt)
	first := strings.Split(content, "\n")[0]
	if !strings.HasPrefix(first, "  leaf") {
		t.Errorf("leaf row = %q, want to start with a space marker, not a branch marker", first)
	}
}

func TestInfoBoxNestedComposesPanel(t *testing.T) {
	tt := theme.DarkTheme()
	rows := []TreeRow{{Key: "root", Children: []TreeRow{{Key: "child", Value: "v"}}}}

	got := InfoBox("Tree", rows, tt, 0)
	want := Panel("Tree", renderRows(rows, tt), tt, 0)
	if got != want {
		t.Errorf("InfoBox(nested) = %q, want %q (Panel wrapping renderRows' output)", got, want)
	}
	if !strings.Contains(got, "root") || !strings.Contains(got, "child") {
		t.Errorf("InfoBox(nested) missing expected rows: %q", got)
	}
}
