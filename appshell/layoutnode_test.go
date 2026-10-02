package appshell

import (
	"strings"
	"testing"

	"fmt"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/widgets"
)

func exact(t *testing.T, out string, wd, ht int) []string {
	t.Helper()
	lines := strings.Split(out, "\n")
	if len(lines) != ht {
		t.Fatalf("%dx%d: %d rows:\n%s", wd, ht, len(lines), out)
	}
	for i, l := range lines {
		if ansi.Width(l) != wd {
			t.Fatalf("%dx%d: row %d is %d wide: %q", wd, ht, i, ansi.Width(l), l)
		}
	}
	return lines
}

var sizes = []layout.Size{{W: 30, H: 8}, {W: 12, H: 3}, {W: 5, H: 1}, {W: 40, H: 20}, {W: 1, H: 1}}

func TestLayoutNodeContentFillsBetweenTitleAndHints(t *testing.T) {
	m := New("My App", 40, 5)
	m.Input.Prompt = "Search: "
	var content []string
	for i := 0; i < 100; i++ {
		content = append(content, fmt.Sprintf("result %d", i))
	}
	m.Content.SetContent(strings.Join(content, "\n"))
	m.Hints = []widgets.Hint{{Key: "q", Action: "quit"}}
	out := ansi.StripANSI(m.LayoutNode().Render(layout.Size{W: 40, H: 12}))
	lines := exact(t, out, 40, 12)
	if !strings.HasPrefix(lines[0], "My App") || !strings.Contains(lines[1], "Search") {
		t.Errorf("title and input first: %q", lines[:2])
	}
	if !strings.Contains(lines[11], "quit") {
		t.Errorf("hints last: %q", lines[11])
	}
	if !strings.HasPrefix(lines[2], "result 0") || !strings.HasPrefix(lines[10], "result 8") {
		t.Errorf("content should fill rows 2 to 10: %q", lines[2:11])
	}
	for _, s := range sizes {
		exact(t, m.LayoutNode().Render(s), s.W, s.H)
	}
}
