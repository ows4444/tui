package widgets

import (
	"strings"

	"github.com/ows4444/tui/theme"
)

// TreeRow is one row of an InfoBox: a key/value pair, optionally with
// nested Children rendered indented beneath it. A row with Children
// renders as a tree branch (treeview's depth-indent/marker convention);
// a row with no Children renders as a flat "key: value" line.
type TreeRow struct {
	Key      string
	Value    string
	Children []TreeRow
}

// InfoBox renders rows inside a Panel: reuses KeyValue's "key: value"
// alignment when every row is flat (no row has Children anywhere), or
// falls back to indented tree rendering — reusing treeview.Model.View's
// "  " (two-space) per-depth indent and ▾/space branch marker — as soon
// as any row is nested. width behaves exactly as it does for Panel.
func InfoBox(title string, rows []TreeRow, t theme.Theme, width int) string {
	return Panel(title, renderRows(rows, t), t, width)
}

func renderRows(rows []TreeRow, t theme.Theme) string {
	if !anyNested(rows) {
		pairs := make([]KV, len(rows))
		for i, r := range rows {
			pairs[i] = KV{Key: r.Key, Value: r.Value}
		}
		return KeyValue(pairs, t)
	}

	g := t.GlyphSet()
	var b strings.Builder
	var walk func(rows []TreeRow, depth int)
	first := true
	walk = func(rows []TreeRow, depth int) {
		for _, r := range rows {
			if !first {
				b.WriteByte('\n')
			}
			first = false

			marker := " "
			if len(r.Children) > 0 {
				marker = g.Expanded
			}
			b.WriteString(strings.Repeat("  ", depth) + marker + " " + rowLabel(r))
			walk(r.Children, depth+1)
		}
	}
	walk(rows, 0)
	return b.String()
}

func rowLabel(r TreeRow) string {
	if r.Value == "" {
		return r.Key
	}
	return r.Key + ": " + r.Value
}

func anyNested(rows []TreeRow) bool {
	for _, r := range rows {
		if len(r.Children) > 0 {
			return true
		}
	}
	return false
}
