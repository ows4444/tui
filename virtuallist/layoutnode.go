package virtuallist

import (
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

// LayoutNode adapts the list to a layout.Node. Measure reports its natural
// size: the widest of the currently visible items by one row per item (the
// full list's height, as a viewport's is its content's). Render shows exactly
// the allotted rows of items from the current offset, with no overscan and
// re-clamped to the new height, calling RenderItem only for the rows it
// shows. The Model is not changed.
func (m Model) LayoutNode() layout.Node { return listNode{m} }

type listNode struct{ m Model }

func (n listNode) itemHeight() int {
	if n.m.ItemHeight < 1 {
		return 1
	}
	return n.m.ItemHeight
}

func (n listNode) Measure(c layout.Constraints) layout.Size {
	if n.m.ItemCount <= 0 || n.m.RenderItem == nil {
		return c.Constrain(layout.Size{})
	}
	w := 0
	end := n.m.offset + n.m.visibleCount()
	if end > n.m.ItemCount {
		end = n.m.ItemCount
	}
	for i := n.m.offset; i < end; i++ {
		if iw := blockWidth(n.m.RenderItem(i)); iw > w {
			w = iw
		}
	}
	return c.Constrain(layout.Size{W: w, H: n.m.ItemCount * n.itemHeight()})
}

func blockWidth(s string) int {
	w := 0
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == '\n' {
			if lw := ansi.Width(s[start:i]); lw > w {
				w = lw
			}
			start = i + 1
		}
	}
	return w
}

func (n listNode) Render(s layout.Size) string {
	if s.W <= 0 || s.H <= 0 {
		return ""
	}
	m := n.m
	ih := n.itemHeight()
	m.Height = (s.H + ih - 1) / ih // items that fill s.H rows
	m.Overscan = 0
	m.setOffset(m.offset)
	return layout.Block(m.View()).Render(s)
}
