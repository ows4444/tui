package widgets

import (
	"strings"

	"github.com/ows4444/tui/theme"
)

// TreeNode is one entry of a static Tree: a label and its children.
type TreeNode struct {
	Label    string
	Children []TreeNode
}

// Tree renders nodes as an indented tree with connector lines, one label per
// line, for output that is printed and not navigated (a directory listing, a
// dependency graph, a plan). A label may span lines; its continuation lines
// stay under the connector. It is the stateless counterpart of treeview.Model.
// An empty nodes renders as "".
func Tree(nodes []TreeNode) string { return TreeWith(nodes, theme.Theme{}) }

// TreeWith is Tree with the connectors drawn from t: ├── └── │ normally, and
// |-- `-- | when t has the ASCII glyph set. Tree is TreeWith on the default theme.
func TreeWith(nodes []TreeNode, t theme.Theme) string {
	if len(nodes) == 0 {
		return ""
	}
	tee, elbow, bar, blank := "├── ", "└── ", "│   ", "    "
	if t.GlyphSet().RuleV == "|" {
		tee, elbow, bar = "|-- ", "`-- ", "|   "
	}
	var lines []string
	var walk func(ns []TreeNode, prefix string)
	walk = func(ns []TreeNode, prefix string) {
		for i, n := range ns {
			last := i == len(ns)-1
			head, tail := tee, bar
			if last {
				head, tail = elbow, blank
			}
			parts := strings.Split(n.Label, "\n")
			lines = append(lines, prefix+head+parts[0])
			for _, p := range parts[1:] {
				lines = append(lines, prefix+tail+p)
			}
			walk(n.Children, prefix+tail)
		}
	}
	walk(nodes, "")
	return strings.Join(lines, "\n")
}
