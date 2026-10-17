package window

import (
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

// LayoutNode adapts the desktop to a layout.Node. It takes the room it is
// given and draws the Background, then each window at its Rect from the back to
// the front, clipped to that room. Measure reports the box that holds every
// window. The Model is not changed.
func (m Model) LayoutNode() layout.Node { return deskNode{m} }

type deskNode struct{ m Model }

func (n deskNode) Measure(c layout.Constraints) layout.Size {
	var s layout.Size
	for _, w := range n.m.Windows {
		s.W, s.H = max(s.W, w.Rect.X+w.Rect.W), max(s.H, w.Rect.Y+w.Rect.H)
	}
	return c.Constrain(s)
}

func (n deskNode) Render(s layout.Size) string {
	if s.W <= 0 || s.H <= 0 {
		return ""
	}
	base := strings.TrimRight(strings.Repeat(strings.Repeat(" ", s.W)+"\n", s.H), "\n")
	if n.m.Background != nil {
		base = n.m.Background.Render(s)
	}
	kids := []layout.Node{layout.Block(base)}
	for i, w := range n.m.Windows {
		if w.Rect.Empty() {
			continue
		}
		kids = append(kids, layout.Layer(i+1, layout.Absolute(w.Rect.X, w.Rect.Y,
			frame{w: w, focused: i == len(n.m.Windows)-1, th: n.m.themed()})))
	}
	return layout.Stack(kids...).Render(s)
}

// frame is one window drawn: border, title bar and content.
type frame struct {
	w       Window
	focused bool
	th      themeView
}

func (f frame) Measure(c layout.Constraints) layout.Size {
	return c.Constrain(layout.Size{W: f.w.Rect.W, H: f.w.Rect.H})
}

func (f frame) Render(s layout.Size) string {
	s = layout.Size{W: min(s.W, f.w.Rect.W), H: min(s.H, f.w.Rect.H)}
	if s.W < 2 || s.H < 2 {
		return layout.Block(strings.TrimRight(strings.Repeat(strings.Repeat(" ", max(s.W, 0))+"\n", max(s.H, 0)), "\n")).Render(s)
	}
	b := f.th.border
	edge, title := f.th.edge, f.th.muted
	if f.focused {
		edge, title = f.th.focus, f.th.focusTitle
	}
	grip := edge
	if f.focused && !f.w.Fixed {
		grip = f.th.focusTitle
	}
	inner := layout.Size{W: s.W - 2, H: s.H - 2}
	rows := make([]string, 0, s.H)
	rows = append(rows, f.top(s.W, b, edge, title))
	body := f.body(inner)
	for i := 0; i < inner.H; i++ {
		rows = append(rows, edge.Render(b.Left)+body[i]+edge.Render(b.Right))
	}
	rows = append(rows, edge.Render(b.BottomLeft)+edge.Render(strings.Repeat(b.Bottom, s.W-2))+grip.Render(b.BottomRight))
	return strings.Join(rows, "\n")
}

// top is the title bar: corner, title, rule, close box, corner.
func (f frame) top(w int, b borderSet, edge, title ansi.Style) string {
	closeBox := ""
	if !f.w.NoClose && w >= defaultMinW {
		closeBox = "[" + f.th.cross + "]"
	}
	room := w - 2 - ansi.Width(closeBox) // between the corners
	text := ""
	if room > 2 && f.w.Title != "" {
		t := ansi.Truncate(strings.NewReplacer("\n", " ", "\r", " ").Replace(ansi.Sanitize(f.w.Title)), room-2)
		text = " " + t + " "
	}
	rule := strings.Repeat(b.Top, max(room-ansi.Width(text), 0))
	out := edge.Render(b.TopLeft) + title.Render(text) + edge.Render(rule)
	if closeBox != "" {
		out += edge.Render(closeBox)
	}
	return out + edge.Render(b.TopRight)
}

// body is the content cut and padded to inner, as inner.H rows.
func (f frame) body(inner layout.Size) []string {
	rows := make([]string, inner.H)
	var lines []string
	if f.w.Content != nil && inner.W > 0 && inner.H > 0 {
		lines = strings.Split(f.w.Content.Render(inner), "\n")
	}
	for i := range rows {
		l := ""
		if i < len(lines) {
			l = ansi.Truncate(lines[i], inner.W)
		}
		rows[i] = l + strings.Repeat(" ", max(inner.W-ansi.Width(l), 0))
	}
	return rows
}
