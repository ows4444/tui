package viewport

import (
	"strings"
	"testing"
)

// AppendLine gives the same Model as SetContent on the joined text.
func TestAppendLineEqualsSetContentOfTheJoinedText(t *testing.T) {
	parts := []string{"alpha", "", "beta\ngamma", "a much wider line than the rest", "end"}
	a := New(8, 3)
	for _, p := range parts {
		a.AppendLine(p)
	}
	b := New(8, 3)
	b.SetContent(strings.Join(parts, "\n"))

	if a.View() != b.View() {
		t.Errorf("view differs:\n%q\n%q", a.View(), b.View())
	}
	if a.maxLineWidth != b.maxLineWidth || len(a.lines) != len(b.lines) {
		t.Errorf("widest %d vs %d, lines %d vs %d", a.maxLineWidth, b.maxLineWidth, len(a.lines), len(b.lines))
	}
	a.GotoBottom()
	b.GotoBottom()
	a.LineRight(100)
	b.LineRight(100)
	if a.View() != b.View() {
		t.Errorf("scrolled view differs:\n%q\n%q", a.View(), b.View())
	}
}

func TestAppendLineToEmptyAndWithEmptyString(t *testing.T) {
	m := New(5, 2)
	m.AppendLine("")
	if len(m.lines) != 1 || m.lines[0] != "" {
		t.Errorf("appending \"\" to an empty viewport gave %q", m.lines)
	}
	m.AppendLine("x")
	if len(m.lines) != 2 || m.lines[1] != "x" {
		t.Errorf("lines = %q", m.lines)
	}
}

// A reader scrolled up stays put as content grows; a reader who follows the
// tail can keep at the bottom.
func TestAppendLineKeepsTheScrollOffsetClamped(t *testing.T) {
	m := New(10, 2)
	for i := 0; i < 5; i++ {
		m.AppendLine("line")
	}
	m.GotoTop()
	m.AppendLine("more")
	if !m.AtTop() {
		t.Error("appending moved a reader who was at the top")
	}
	m.GotoBottom()
	bottom := m.yOffset
	m.AppendLine("tail")
	if m.yOffset != bottom {
		t.Errorf("yOffset changed from %d to %d on append", bottom, m.yOffset)
	}
	if m.AtBottom() {
		t.Error("the viewport should no longer be at the bottom after the content grew")
	}
	m.GotoBottom()
	if !m.AtBottom() {
		t.Error("GotoBottom after append did not reach the bottom")
	}
}

func TestAppendLineWidensTheScrollRange(t *testing.T) {
	m := New(4, 1)
	m.AppendLine("ab")
	m.LineRight(50)
	if m.xOffset != 0 {
		t.Fatalf("nothing wider than the viewport, xOffset = %d", m.xOffset)
	}
	m.AppendLine("0123456789")
	m.LineRight(50)
	if m.xOffset != 6 { // 10 wide, 4 shown
		t.Errorf("xOffset = %d, want 6", m.xOffset)
	}
}
