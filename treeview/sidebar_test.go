package treeview

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

func sidebarTree() Model {
	return New(Node{
		Label: "root",
		Children: []Node{
			{Label: "src", Children: []Node{
				{Label: "main.go"},
			}},
			{Label: "README.md"},
		},
	})
}

func expand(m Model, at int) Model {
	m.SetCursor(at)
	m, _ = m.Update(tui.Key{Type: tui.KeyEnter})
	return m
}

func TestSidebarWrapsTreeviewInPanel(t *testing.T) {
	tt := theme.DarkTheme()
	tv := sidebarTree()

	got := Sidebar("Files", tv, "", nil, nil, tt, 0)
	want := widgets.Panel("Files", "▸ root", tt, 0)
	if got != want {
		t.Errorf("Sidebar (collapsed, no decoration) = %q, want %q", got, want)
	}
}

func TestSidebarActiveKeyHighlightsIndependentlyOfCursor(t *testing.T) {
	tt := theme.DarkTheme()
	tv := expand(sidebarTree(), 0) // expand root; cursor stays at "root" (index 0)

	got := Sidebar("Files", tv, "0.1", nil, nil, tt, 0)
	activeStyle := tt.ResolvedStates().Selected.Bold()

	if !strings.Contains(got, activeStyle.Render("    README.md")) {
		t.Errorf("Sidebar with activeKey=0.1 does not contain a highlighted README.md row: %q", got)
	}
	if strings.Contains(got, activeStyle.Render("▾ root")) {
		t.Error("Sidebar highlighted the cursor row even though activeKey pointed elsewhere")
	}
}

func TestSidebarNoActiveKeyMatchHighlightsNothing(t *testing.T) {
	tt := theme.DarkTheme()
	tv := sidebarTree()
	plain := Sidebar("Files", tv, "does-not-exist-anywhere", nil, nil, tt, 0)
	unhighlighted := Sidebar("Files", tv, "", nil, nil, tt, 0)
	if plain != unhighlighted {
		t.Errorf("activeKey matching nothing should render identically to no activeKey:\n%q\n%q", plain, unhighlighted)
	}
}

func TestSidebarIconDecoration(t *testing.T) {
	tt := theme.DarkTheme()
	tv := sidebarTree()

	withIcon := Sidebar("Files", tv, "", map[string]string{"0": "📁"}, nil, tt, 0)
	if !strings.Contains(withIcon, "📁 root") {
		t.Errorf("Sidebar with an icon for path %q missing icon in output: %q", "0", withIcon)
	}

	noIcon := Sidebar("Files", tv, "", nil, nil, tt, 0)
	if strings.Contains(noIcon, "📁") {
		t.Error("Sidebar with no icons registered should render no icon glyph")
	}
}

func TestSidebarBadgeDecorationWithinWidth(t *testing.T) {
	tt := theme.DarkTheme()
	tv := sidebarTree()

	got := Sidebar("Files", tv, "", nil, map[string]string{"0": "3"}, tt, 0)
	if !strings.Contains(got, "root 3") {
		t.Errorf("Sidebar with a badge for path %q missing badge in output: %q", "0", got)
	}

	fixed := Sidebar("Files", tv, "", nil, map[string]string{"0": "a-very-long-badge-that-would-overflow"}, tt, 20)
	for _, line := range strings.Split(fixed, "\n") {
		if w := ansi.Width(line); w > 20 {
			t.Errorf("line %q has width %d, want <= 20", line, w)
		}
	}
}

func TestSidebarEmptyTreeNoPanic(t *testing.T) {
	tt := theme.DarkTheme()
	empty := New()

	got := Sidebar("Files", empty, "", nil, nil, tt, 0)
	want := widgets.Panel("Files", "", tt, 0)
	if got != want {
		t.Errorf("Sidebar(empty tree) = %q, want %q", got, want)
	}
}

func TestSidebarWidthBoundedAcrossCombinations(t *testing.T) {
	tt := theme.DarkTheme()
	tv := expand(sidebarTree(), 0)
	tv = expand(tv, 1) // "src" is now visible at index 1; expand it too

	icons := map[string]string{"0": "📁", "0.0": "📂", "0.0.0": "📄"}
	badges := map[string]string{"0": "12", "0.1": "new"}

	for _, width := range []int{10, 20, 40} {
		got := Sidebar("Files", tv, "0.0.0", icons, badges, tt, width)
		for _, line := range strings.Split(got, "\n") {
			if w := ansi.Width(line); w > width {
				t.Errorf("width=%d: line %q has width %d", width, line, w)
			}
		}
	}
}
