package treeview

import (
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

// Sidebar renders tv's visible rows (as Model.View does: an
// indent-and-marker tree, delegating all flatten/expand-state/cursor
// logic to tv itself via Model.VisibleRows) inside a widgets.Panel, with
// optional per-row icon and badge decoration and a highlight driven by
// activeKey rather than tv's own keyboard cursor — activeKey identifies
// which row's content is currently shown elsewhere (e.g. the open file in
// a file-tree-plus-editor layout), which can differ from where the
// keyboard cursor happens to be.
//
// icons and badges are looked up by a row's Path (see VisibleRow
// and Model's doc comment on path stability); a path with no entry in
// icons renders with no icon glyph, and likewise for badges. width
// behaves exactly as it does for Panel.
func Sidebar(title string, tv Model, activeKey string, icons, badges map[string]string, t theme.Theme, width int) string {
	rows := tv.VisibleRows()
	cw, auto := sidebarContentWidth(width)

	activeStyle := t.ResolvedStates().Selected.Bold()
	g := t.GlyphSet()

	lines := make([]string, len(rows))
	for i, r := range rows {
		marker := " "
		if r.HasChildren {
			if tv.IsExpanded(r.Path) {
				marker = g.Expanded
			} else {
				marker = g.Collapsed
			}
		}

		label := r.Label
		if icon, ok := icons[r.Path]; ok {
			label = icon + " " + label
		}
		line := strings.Repeat("  ", r.Depth) + marker + " " + label
		if badge, ok := badges[r.Path]; ok {
			line += " " + badge
		}

		if !auto {
			line = ansi.Truncate(line, cw)
		}
		if r.Path == activeKey {
			line = activeStyle.Render(line)
		}
		lines[i] = line
	}

	return widgets.Panel(title, strings.Join(lines, "\n"), t, width)
}

// sidebarFrame is the padding+border overhead widgets.Panel adds around its
// content: 1 column of padding plus 1 border character on each side.
const sidebarFrame = 4

// sidebarContentWidth turns Sidebar's total width into the content width
// inside the Panel. A width <= 0 means size-to-content, reported as auto.
func sidebarContentWidth(width int) (cw int, auto bool) {
	if width <= 0 {
		return 0, true
	}
	return max(width-sidebarFrame, 0), false
}
