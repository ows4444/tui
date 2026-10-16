package widgets

import (
	"testing"

	"github.com/ows4444/tui/theme"
)

func TestTreeDrawsConnectors(t *testing.T) {
	got := Tree([]TreeNode{
		{Label: "src", Children: []TreeNode{
			{Label: "main.go"},
			{Label: "util", Children: []TreeNode{{Label: "a.go"}, {Label: "b.go"}}},
		}},
		{Label: "README.md"},
	})
	want := "├── src\n" +
		"│   ├── main.go\n" +
		"│   └── util\n" +
		"│       ├── a.go\n" +
		"│       └── b.go\n" +
		"└── README.md"
	if got != want {
		t.Errorf("Tree =\n%s\nwant\n%s", got, want)
	}
}

func TestTreeKeepsMultilineLabelsUnderTheirConnector(t *testing.T) {
	got := Tree([]TreeNode{{Label: "a\nb", Children: []TreeNode{{Label: "c"}}}, {Label: "d"}})
	want := "├── a\n│   b\n│   └── c\n└── d"
	if got != want {
		t.Errorf("Tree =\n%s\nwant\n%s", got, want)
	}
}

func TestTreeEmptyAndASCII(t *testing.T) {
	if Tree(nil) != "" {
		t.Error("an empty tree is not empty")
	}
	got := TreeWith([]TreeNode{{Label: "a", Children: []TreeNode{{Label: "b"}}}, {Label: "c"}}, theme.DarkTheme().ASCII())
	want := "|-- a\n|   `-- b\n`-- c"
	if got != want {
		t.Errorf("ASCII tree =\n%s\nwant\n%s", got, want)
	}
}
