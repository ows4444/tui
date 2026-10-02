package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

// LayoutInspector is implemented by a root Model that draws through the
// layout package and can hand the inspector the tree it drew, so the overlay
// can list the named rectangles. Name nodes with layout.Named.
type LayoutInspector interface {
	InspectLayout() layout.Node
}

// FocusInspector is implemented by a root Model that can name the widget that
// has focus, for the inspector overlay.
type FocusInspector interface {
	FocusedID() string
}

// The default keys of the inspector panes, used for an InspectorKeys field left
// as "".
const (
	// DefaultInspectorKey toggles the info panel.
	DefaultInspectorKey = "f12"
	// DefaultInspectorLayoutKey toggles the layout outlines.
	DefaultInspectorLayoutKey = "f11"
	// DefaultInspectorMessagesKey toggles the message pane.
	DefaultInspectorMessagesKey = "f10"
	// DefaultInspectorModelKey toggles the model pane.
	DefaultInspectorModelKey = "f9"
)

// InspectorOff, as a field of InspectorKeys, leaves that pane off.
const InspectorOff = "off"

// InspectorKeys names the key that toggles each inspector pane, as Key.String
// reports it ("f12", "ctrl+i"). A field left as "" takes that pane's default key
// (DefaultInspectorKey and its siblings); InspectorOff leaves the pane out.
type InspectorKeys struct {
	// Panel toggles the info panel over the top-right of the view: the named
	// rectangles of the tree (the root Model implements LayoutInspector), the
	// focused id (FocusInspector), and the last frame's size in bytes and time
	// to draw. Default f12.
	Panel string
	// Layout outlines every named rectangle of the tree with a box whose top
	// edge carries the node's name and size ("name WxH"), drawn over the view
	// without covering its content. Default f11. When the panel is also on, it
	// is drawn over the outlines.
	Layout string
	// Messages draws, over the top-left of the view, the most recent messages
	// Update received, newest last, each with its type, value and how long
	// Update took. The last 200 are kept, and are only recorded while this pane
	// is on; the keys that toggle inspector panes are not recorded. Default f10.
	Messages string
	// Model draws, over the bottom-left of the view, a %#v dump of the root
	// Model, wrapped and cut to fit. Default f9.
	Model string
}

// WithInspector turns on the devtool overlays named by keys: a zero
// InspectorKeys turns on all four at their default keys. Pressing a pane's key
// shows it and pressing it again removes it; the key is consumed by the Program
// and never reaches Update. Without this option nothing is tracked and nothing
// is drawn.
func WithInspector(keys InspectorKeys) ProgramOption {
	pick := func(k, def string) string {
		switch k {
		case "":
			return def
		case InspectorOff:
			return ""
		}
		return k
	}
	return func(p *Program) {
		p.inspectorKey = pick(keys.Panel, DefaultInspectorKey)
		p.outlineKey = pick(keys.Layout, DefaultInspectorLayoutKey)
		p.msgKey = pick(keys.Messages, DefaultInspectorMessagesKey)
		p.modelKey = pick(keys.Model, DefaultInspectorModelKey)
	}
}

// inspectorKeyPressed toggles the overlay when msg is an inspector key and
// reports whether it was (the caller then redraws and drops msg).
func (p *Program) inspectorKeyPressed(msg Msg) bool {
	if (p.inspectorKey == "" && p.outlineKey == "" && p.msgKey == "" && p.modelKey == "") || p.accessible {
		return false
	}
	k, ok := msg.(Key)
	if !ok || k.Action == KeyRelease {
		return false
	}
	switch s := k.String(); {
	case p.inspectorKey != "" && s == p.inspectorKey:
		p.inspectorOn = !p.inspectorOn
	case p.outlineKey != "" && s == p.outlineKey:
		p.outlineOn = !p.outlineOn
	case p.msgKey != "" && s == p.msgKey:
		p.msgOn = !p.msgOn
	case p.modelKey != "" && s == p.modelKey:
		p.modelOn = !p.modelOn
	default:
		return false
	}
	return true
}

// noteFrame records the frame just written for the inspector.
func (p *Program) noteFrame(bytes int, start time.Time) {
	if p.inspectorKey == "" || start.IsZero() {
		return
	}
	p.lastFrameBytes, p.lastFrameDur = bytes, time.Since(start)
}

// withInspector composites the overlay on view when it is showing.
func (p *Program) withInspector(view string) string {
	if p.outlineOn {
		view = p.withOutlines(view)
	}
	if (p.msgOn || p.modelOn) && p.height > 0 {
		// Overlay clips to the view's rows: give a short view the screen's.
		if n := strings.Count(view, "\n") + 1; n < p.height {
			view += strings.Repeat("\n", p.height-n)
		}
	}
	if p.msgOn {
		view = layout.Overlay(view, p.boxed(p.msgRows()), 0, 0)
	}
	if p.modelOn {
		panel := p.boxed(p.modelRows())
		view = layout.Overlay(view, panel, 0, max(0, p.height-strings.Count(panel, "\n")-1))
	}
	if !p.inspectorOn {
		return view
	}
	panel := p.inspectorPanel()
	w := 0
	for _, l := range strings.Split(panel, "\n") {
		w = max(w, p.Measurer().Width(l))
	}
	x := 0
	if p.width > w {
		x = p.width - w
	}
	return layout.Overlay(view, panel, x, 0)
}

// inspectorPanel is the overlay text: a bordered box of fixed-width rows.
func (p *Program) inspectorPanel() string {
	rows := []string{
		"inspector (" + p.inspectorKey + " closes)",
		fmt.Sprintf("last frame: %d bytes, %d us", p.lastFrameBytes, p.lastFrameDur.Microseconds()),
	}
	focus := "-"
	if fi, ok := find[FocusInspector](p.model); ok {
		if id := fi.FocusedID(); id != "" {
			focus = id
		}
	}
	rows = append(rows, "focus: "+focus)
	if li, ok := find[LayoutInspector](p.model); ok && li.InspectLayout() != nil {
		rows = append(rows, "named rects:")
		named := 0
		for _, pl := range layout.Rects(li.InspectLayout(), layout.Size{W: p.width, H: p.height}) {
			if pl.Name == "" {
				continue
			}
			named++
			rows = append(rows, fmt.Sprintf("  %s %d,%d %dx%d", pl.Name, pl.Rect.X, pl.Rect.Y, pl.Rect.W, pl.Rect.H))
		}
		if named == 0 {
			rows = append(rows, "  (none: use layout.Named)")
		}
	}
	return p.boxed(rows)
}

// boxed draws rows in a bordered box of fixed-width rows.
func (p *Program) boxed(rows []string) string {
	maxW := 0
	for i, r := range rows {
		r = ansi.Sanitize(r)
		if p.width > 4 {
			r = p.Measurer().Truncate(r, p.width-4)
		}
		rows[i] = r
		maxW = max(maxW, p.Measurer().Width(r))
	}
	bar := "+" + strings.Repeat("-", maxW+2) + "+"
	out := []string{bar}
	for _, r := range rows {
		out = append(out, "| "+r+strings.Repeat(" ", maxW-p.Measurer().Width(r))+" |")
	}
	return strings.Join(append(out, bar), "\n")
}

// inspectorMsgCap is how many messages the inspector keeps.
const inspectorMsgCap = 200

// inspectorDumpCap bounds the %#v text the model pane formats.
const inspectorDumpCap = 16 << 10

// msgEntry is one logged message: what it was and how long Update took.
type msgEntry struct {
	n    int // 1-based sequence number
	desc string
	dur  time.Duration
}

// msgRing holds the last inspectorMsgCap messages.
type msgRing struct {
	buf   []msgEntry
	total int
}

func (r *msgRing) add(msg Msg, d time.Duration) {
	r.total++
	e := msgEntry{n: r.total, desc: fmt.Sprintf("%T %v", msg, msg), dur: d}
	if len(r.buf) < inspectorMsgCap {
		r.buf = append(r.buf, e)
		return
	}
	r.buf[(r.total-1)%inspectorMsgCap] = e
}

// entries returns the logged messages, oldest first.
func (r *msgRing) entries() []msgEntry {
	if len(r.buf) < inspectorMsgCap {
		return append([]msgEntry(nil), r.buf...)
	}
	k := r.total % inspectorMsgCap
	return append(append([]msgEntry(nil), r.buf[k:]...), r.buf[:k]...)
}

// msgRows is the message pane: newest last, as many as fit the screen.
func (p *Program) msgRows() []string {
	all := p.msgLog.entries()
	rows := []string{fmt.Sprintf("messages (%s closes) last %d of %d", p.msgKey, len(all), p.msgLog.total)}
	room := max(1, p.height-3)
	if len(all) > room-1 {
		all = all[len(all)-(room-1):]
	}
	for _, e := range all {
		d := strings.NewReplacer("\n", " ", "\r", " ").Replace(e.desc)
		rows = append(rows, fmt.Sprintf("#%d %s  %d us", e.n, p.Measurer().Truncate(ansi.Sanitize(d), 40), e.dur.Microseconds()))
	}
	return rows
}

// modelRows is the model pane: the %#v dump of the root Model.
func (p *Program) modelRows() []string {
	dump := fmt.Sprintf("%#v", p.model)
	if len(dump) > inspectorDumpCap {
		dump = dump[:inspectorDumpCap] + "..."
	}
	width := max(10, p.width-8)
	rows := []string{"model (" + p.modelKey + " closes)"}
	room := max(1, p.height/2)
	for _, line := range strings.Split(dump, "\n") {
		line = ansi.Sanitize(line)
		for line != "" {
			head := p.Measurer().Truncate(line, width)
			if head == "" {
				break
			}
			rows = append(rows, head)
			line = line[len(head):]
		}
	}
	if len(rows) > room {
		rows = append(rows[:room-1], "...")
	}
	return rows
}

// withOutlines draws a box around every named rectangle of the model's layout
// tree over view, labelled "name WxH" on its top edge. Only the edge cells are
// replaced, so the content inside stays visible.
func (p *Program) withOutlines(view string) string {
	li, ok := find[LayoutInspector](p.model)
	if !ok || li.InspectLayout() == nil {
		return view
	}
	for _, pl := range layout.Rects(li.InspectLayout(), layout.Size{W: p.width, H: p.height}) {
		if pl.Name == "" || pl.Rect.Empty() {
			continue
		}
		view = p.outline(view, pl.Rect, ansi.Sanitize(pl.Name))
	}
	return view
}

// outline draws r's box on view: corners "+", edges "-" and "|", and the label
// "name WxH" at the left of the top edge, cut to fit. A rectangle one cell
// tall or wide gets as much of the outline as fits.
func (p *Program) outline(view string, r layout.Rect, name string) string {
	m := p.Measurer()
	if r.W == 1 {
		for y := 0; y < r.H; y++ {
			view = layout.Overlay(view, "|", r.X, r.Y+y)
		}
		return view
	}
	top := m.Truncate(fmt.Sprintf("+%s %dx%d", name, r.W, r.H), r.W-1)
	top += strings.Repeat("-", r.W-1-m.Width(top)) + "+"
	view = layout.Overlay(view, top, r.X, r.Y)
	if r.H == 1 {
		return view
	}
	for y := 1; y < r.H-1; y++ {
		view = layout.Overlay(view, "|", r.X, r.Y+y)
		view = layout.Overlay(view, "|", r.X+r.W-1, r.Y+y)
	}
	return layout.Overlay(view, "+"+strings.Repeat("-", r.W-2)+"+", r.X, r.Y+r.H-1)
}
