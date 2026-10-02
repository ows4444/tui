package viewport

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

func doc(n int) string {
	lines := make([]string, n)
	for i := range lines {
		lines[i] = fmt.Sprintf("line %02d ....................", i)
	}
	return strings.Join(lines, "\n")
}

func exactLines(t *testing.T, out string, w, h int) []string {
	t.Helper()
	lines := strings.Split(out, "\n")
	if len(lines) != h {
		t.Fatalf("%d rows, want %d", len(lines), h)
	}
	for i, l := range lines {
		if ansi.Width(l) != w {
			t.Fatalf("row %d width %d, want %d: %q", i, ansi.Width(l), w, l)
		}
	}
	return lines
}

func TestLayoutNodeMeasureAndRenderUseTheAllottedSize(t *testing.T) {
	m := New(5, 2) // its own size is small on purpose
	m.SetContent(doc(30))
	n := m.LayoutNode()

	if got := n.Measure(layout.Unconstrained()); got != (layout.Size{W: 28, H: 30}) {
		t.Errorf("natural Measure = %v", got)
	}
	if got := n.Measure(layout.Loose(layout.Size{W: 10, H: 4})); got != (layout.Size{W: 10, H: 4}) {
		t.Errorf("constrained Measure = %v", got)
	}

	lines := exactLines(t, n.Render(layout.Size{W: 12, H: 4}), 12, 4)
	if strings.TrimRight(lines[0], " ") != "line 00 ...." || strings.TrimRight(lines[3], " ") != "line 03 ...." {
		t.Errorf("window = %q", lines)
	}
	if m.Width != 5 || m.Height != 2 {
		t.Errorf("rendering resized the Model to %dx%d", m.Width, m.Height)
	}
}

func TestLayoutNodeKeepsScrollOffsetAndReclamps(t *testing.T) {
	m := New(10, 3)
	m.SetContent(doc(30))
	m.LineDown(10)
	lines := exactLines(t, m.LayoutNode().Render(layout.Size{W: 10, H: 3}), 10, 3)
	if !strings.HasPrefix(lines[0], "line 10") {
		t.Errorf("scroll offset lost: %q", lines[0])
	}
	// A taller slot than the remaining content re-clamps rather than blanking.
	m.GotoBottom()
	lines = exactLines(t, m.LayoutNode().Render(layout.Size{W: 10, H: 12}), 10, 12)
	if !strings.HasPrefix(lines[0], "line 18") { // 30 - 12
		t.Errorf("offset not re-clamped to the taller size: %q", lines[0])
	}
}

func TestLayoutNodeDegenerateSizesAndEmptyContent(t *testing.T) {
	if got := New(5, 5).LayoutNode().Render(layout.Size{W: 0, H: 3}); got != "" {
		t.Errorf("zero width = %q", got)
	}
	empty := New(5, 5).LayoutNode()
	if got := empty.Measure(layout.Unconstrained()); got != (layout.Size{}) {
		t.Errorf("empty Measure = %v", got)
	}
	exactLines(t, empty.Render(layout.Size{W: 4, H: 2}), 4, 2)
}
