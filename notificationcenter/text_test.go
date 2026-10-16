package notificationcenter

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/widgets"
)

func TestNotificationTextKeepsItsOwnStylesAndTakesTheVariantColourElsewhere(t *testing.T) {
	m := New(3, 30)
	m.Push(Notification{
		Message: "ignored",
		Text:    ansi.NewText("Build ", ansi.Style{}).Append("failed", ansi.NewStyle().Bold()),
		Variant: widgets.VariantError,
	})
	out := m.Render("a\nb\nc\nd\ne\nf")
	if strings.Contains(out, "ignored") {
		t.Error("Message was drawn although Text is set")
	}
	if !strings.Contains(out, "\x1b[1mfailed") {
		t.Errorf("the span's own style was lost: %q", out)
	}
	if !strings.Contains(out, "Build") {
		t.Errorf("text missing: %q", out)
	}
	if got := m.Linearize(); !strings.Contains(got, "error 1 of 1: Build failed") {
		t.Errorf("Linearize = %q", got)
	}
	node := ansi.StripANSI(m.LayoutNode().Render(layout.Size{W: 30, H: 6}))
	if !strings.Contains(node, "Build failed") {
		t.Errorf("LayoutNode = %q", node)
	}
}

func TestNotificationTextWrapsAtWidthAndMessageStillWorks(t *testing.T) {
	m := New(3, 12)
	m.Push(Notification{Text: ansi.NewText("one two three four", ansi.Style{})})
	m.Push(Notification{Message: "plain message here"})
	out := ansi.StripANSI(m.Render(strings.Repeat(strings.Repeat(" ", 30)+"\n", 10)))
	if !strings.Contains(out, "one two") || !strings.Contains(out, "plain") {
		t.Errorf("render:\n%s", out)
	}
}
