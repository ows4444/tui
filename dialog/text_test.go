package dialog

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

func layoutOf(m Model) string { return m.LayoutNode().Render(layout.Size{W: 40, H: 8}) }

func TestMessageTextIsShownInsteadOfMessageWithItsStyles(t *testing.T) {
	m := New("Delete", "ignored message")
	m.MessageText = ansi.NewText("This is ", ansi.Style{}).Append("permanent", ansi.NewStyle().Bold())
	view := m.Render("base\nbase\nbase\nbase\nbase\nbase")
	if strings.Contains(view, "ignored") {
		t.Error("Message was drawn although MessageText is set")
	}
	if !strings.Contains(view, "\x1b[1mpermanent") {
		t.Errorf("the bold span lost its style: %q", view)
	}
	if got := m.Linearize(); got != "Dialog: Delete\nThis is permanent" {
		t.Errorf("Linearize = %q", got)
	}
	if out := ansi.StripANSI(layoutOf(m)); !strings.Contains(out, "This is permanent") {
		t.Errorf("LayoutNode = %q", out)
	}
}

func TestMessageTextWrapsAtWidthKeepingStyles(t *testing.T) {
	m := New("T", "")
	m.Width = 10
	m.MessageText = ansi.NewText("plain ", ansi.Style{}).Append("bold words here", ansi.NewStyle().Bold())
	body := ansi.StripANSI(m.Render(strings.Repeat(strings.Repeat(" ", 30)+"\n", 8)))
	if !strings.Contains(body, "plain bold") || !strings.Contains(body, "words here") {
		t.Errorf("not wrapped at 10:\n%s", body)
	}
}

func TestMessageTextEmptyLeavesMessageAlone(t *testing.T) {
	m := New("T", "hello there")
	if !strings.Contains(ansi.StripANSI(m.Render("x\nx\nx\nx\nx\nx\nx")), "hello there") {
		t.Error("Message stopped working")
	}
}
