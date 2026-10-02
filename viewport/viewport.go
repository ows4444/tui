// Package viewport is a scrollable window onto content taller than it,
// built on top of the root tui package the same way textinput is.
package viewport

import (
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/cellbuf"
	"github.com/ows4444/tui/keymap"
)

// Model is a fixed-size scrollable window onto SetContent's lines.
type Model struct {
	Width  int
	Height int

	// Raw, when true, makes SetContent and AppendLine keep text unchanged.
	// By default they expand tabs and strip every escape sequence and
	// control character except SGR styling (ansi.SanitizeKeepSGR).
	Raw bool

	// KeyMap holds the keys for each scroll action. New fills it with
	// DefaultKeyMap; a Model built as a struct literal with a zero KeyMap
	// behaves as if it held DefaultKeyMap. Mouse wheel scrolling is not
	// key-driven and is not part of it.
	KeyMap KeyMap

	// SoftWrap, when true, wraps lines longer than Width onto further rows by
	// display width (never splitting a grapheme cluster) instead of clipping
	// them. Scrolling, offsets, AtBottom and ScrollPercent then count visual
	// rows; LineCount still counts content lines. Horizontal scrolling is off.
	// Toggling it or changing Width is picked up by the next Update or content
	// change; until then reads rebuild the wrap on the fly.
	SoftWrap bool

	lines        []string // backing store; the content is lines[head:]
	head         int      // count of dropped-from-the-front entries not yet compacted away
	yOffset      int
	xOffset      int
	maxLineWidth int // widest line's ansi.Width across all of lines, cached at SetContent

	// soft-wrap row cache, valid while wwidth == Width and wcovered == len(lines)
	wrows    []vrow
	wwidth   int
	wcovered int
	wfrom    int

	// search state
	query  string
	cur    matchRef
	curSet bool
}

// New returns a Model sized to width x height. Content is set separately
// via SetContent.
func New(width, height int) Model {
	return Model{Width: width, Height: height, KeyMap: DefaultKeyMap()}
}

// KeyMap names the keys of each action of a Model.
type KeyMap struct {
	Up       keymap.Binding // scroll up one line
	Down     keymap.Binding // scroll down one line
	PageUp   keymap.Binding // scroll up one page
	PageDown keymap.Binding // scroll down one page
	Top      keymap.Binding // scroll to the top
	Bottom   keymap.Binding // scroll to the bottom

	// NextMatch and PrevMatch move between search matches while a query is set
	// (see Search). When both are empty they default to "n" and "N". They are
	// not part of Bindings, which lists the scroll actions.
	NextMatch keymap.Binding
	PrevMatch keymap.Binding
}

// DefaultKeyMap returns the keys a Model used before KeyMap existed.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up:       keymap.NewBinding("scroll up", "up"),
		Down:     keymap.NewBinding("scroll down", "down"),
		PageUp:   keymap.NewBinding("page up", "pgup"),
		PageDown: keymap.NewBinding("page down", "pgdown"),
		Top:      keymap.NewBinding("top", "home"),
		Bottom:   keymap.NewBinding("bottom", "end"),

		NextMatch: defaultNextMatch(),
		PrevMatch: defaultPrevMatch(),
	}
}

func (km KeyMap) list() []keymap.Binding {
	return []keymap.Binding{km.Up, km.Down, km.PageUp, km.PageDown, km.Top, km.Bottom}
}

func (m Model) keys() KeyMap {
	for _, b := range m.KeyMap.list() {
		if len(b.Keys) > 0 {
			return m.KeyMap
		}
	}
	return DefaultKeyMap()
}

// Bindings returns the scroll actions the viewport currently honours, with
// descriptions, for help text.
func (m Model) Bindings() []keymap.Binding { return m.keys().list() }

// clean applies the default sanitising unless Raw is set.
func (m Model) clean(s string) string {
	if m.Raw {
		return s
	}
	return ansi.SanitizeKeepSGR(ansi.ExpandTabs(s))
}

// plainWidth reports whether every line of s is printable ASCII and, if so,
// the width of the widest one. For such text clean is the identity.
func plainWidth(s string) (int, bool) {
	widest := 0
	for rest := s; ; {
		line, next, more := strings.Cut(rest, "\n")
		w, ok := ansi.PlainASCIIWidth(line)
		if !ok {
			return 0, false
		}
		if w > widest {
			widest = w
		}
		if !more {
			return widest, true
		}
		rest = next
	}
}

// SetContent replaces the content, re-clamping both scroll offsets if the
// new content is shorter, or narrower, than before.
func (m *Model) SetContent(s string) {
	if !m.Raw {
		// Printable-ASCII text, the common case for logs, needs no
		// sanitizing, and checking that is the same pass that measures the
		// widest line, so it costs nothing over Raw. Anything else takes the
		// full clean below.
		if w, ok := plainWidth(s); ok {
			m.lines = strings.Split(s, "\n")
			m.head = 0
			m.maxLineWidth = w
			m.curSet = false
			m.sync(true)
			m.setYOffset(m.yOffset)
			m.setXOffset(m.xOffset)
			return
		}
	}
	s = m.clean(s)
	m.lines = strings.Split(s, "\n")
	m.head = 0
	m.maxLineWidth = 0
	for _, l := range m.lines {
		if w := ansi.Width(l); w > m.maxLineWidth {
			m.maxLineWidth = w
		}
	}
	m.curSet = false
	m.sync(true)
	m.setYOffset(m.yOffset)
	m.setXOffset(m.xOffset)
}

// AppendLine adds s to the end of the content without touching the lines
// already there, so appending n lines costs O(n) in total, where calling
// SetContent with the whole text each time costs O(n^2). s may contain "\n";
// each part becomes its own line, exactly as SetContent would split it. The
// scroll offsets are re-clamped like SetContent (a reader who is scrolled up
// stays where they are; use AtBottom and GotoBottom to follow the tail).
//
// It grows the Model's line slice in place, like append: do not append to two
// copies of the same Model and expect them to stay independent.
func (m *Model) AppendLine(s string) {
	s = m.clean(s)
	first := len(m.lines)
	m.lines = append(m.lines, strings.Split(s, "\n")...)
	for _, l := range m.lines[first:] {
		if w := ansi.Width(l); w > m.maxLineWidth {
			m.maxLineWidth = w
		}
	}
	m.sync(false)
	m.setYOffset(m.yOffset)
	m.setXOffset(m.xOffset)
}

// LineCount returns the number of content lines currently held.
func (m Model) LineCount() int { return len(m.lines) - m.head }

// TrimFront drops the oldest lines so at most max remain, and returns how many
// were dropped (0 if max is negative or the content already fits). The
// vertical offset moves up by the dropped count, clamped at 0, so a reader
// scrolled back keeps seeing the same content; a view pinned to the bottom
// stays pinned. Dropping is amortised O(1) per line: lines are skipped by
// advancing an index, and the backing slice is compacted only once the
// skipped part outgrows the live part. At compaction the widest-line cache is
// recomputed; between compactions it may still count a dropped line, so
// horizontal scroll range can briefly be a little wider than the content.
func (m *Model) TrimFront(max int) int {
	if max < 0 {
		return 0
	}
	n := m.LineCount() - max
	if n <= 0 {
		return 0
	}
	// Copy-on-write: m.lines may be shared with copies of m, so the skipped
	// lines are neither cleared nor compacted in place; compaction allocates
	// a fresh slice, which also releases the dropped strings for GC.
	rowsBefore := 0
	if m.wrapOn() {
		rowsBefore = len(m.vrows())
	}
	m.head += n
	dropped := n // rows scrolled off the top: lines, or visual rows when wrapping
	if m.wrapOn() {
		dropped = rowsBefore - len(m.vrows())
	}
	compact := m.head > m.LineCount()
	if compact {
		if m.curSet {
			m.cur.line -= m.head
			m.curSet = m.cur.line >= 0
		}
		m.lines = append([]string(nil), m.lines[m.head:]...)
		m.head = 0
		m.maxLineWidth = 0
		for _, l := range m.lines {
			if w := ansi.Width(l); w > m.maxLineWidth {
				m.maxLineWidth = w
			}
		}
	}
	m.sync(compact)
	m.yOffset -= dropped
	m.setYOffset(m.yOffset)
	m.setXOffset(m.xOffset)
	return n
}

func (m Model) maxYOffset() int {
	max := m.total() - m.Height
	if max < 0 {
		return 0
	}
	return max
}

// maxXOffset is how far right the viewport can scroll: the widest line's
// width beyond what Width already shows, or 0 if every line already fits.
func (m Model) maxXOffset() int {
	max := m.maxLineWidth - m.Width
	if max < 0 || m.wrapOn() {
		return 0
	}
	return max
}

func (m *Model) setYOffset(y int) { m.yOffset = clamp(y, 0, m.maxYOffset()) }
func (m *Model) setXOffset(x int) { m.xOffset = clamp(x, 0, m.maxXOffset()) }

// LineUp scrolls up n lines, clamped to the top.
func (m *Model) LineUp(n int) { m.setYOffset(m.yOffset - n) }

// LineDown scrolls down n lines, clamped to the bottom.
func (m *Model) LineDown(n int) { m.setYOffset(m.yOffset + n) }

// LineLeft scrolls left n columns, clamped to the left edge.
func (m *Model) LineLeft(n int) { m.setXOffset(m.xOffset - n) }

// LineRight scrolls right n columns, clamped to the rightmost column that
// still shows content (see maxXOffset).
func (m *Model) LineRight(n int) { m.setXOffset(m.xOffset + n) }

// PageUp scrolls up one Height's worth of lines.
func (m *Model) PageUp() { m.LineUp(m.Height) }

// PageDown scrolls down one Height's worth of lines.
func (m *Model) PageDown() { m.LineDown(m.Height) }

// HalfPageUp scrolls up half a Height's worth of lines.
func (m *Model) HalfPageUp() { m.LineUp(m.Height / 2) }

// HalfPageDown scrolls down half a Height's worth of lines.
func (m *Model) HalfPageDown() { m.LineDown(m.Height / 2) }

// GotoTop scrolls to the very top of the content.
func (m *Model) GotoTop() { m.setYOffset(0) }

// GotoBottom scrolls to the very bottom of the content.
func (m *Model) GotoBottom() { m.setYOffset(m.maxYOffset()) }

// AtTop reports whether the viewport is scrolled to the top.
func (m Model) AtTop() bool { return m.yOffset <= 0 }

// AtBottom reports whether the viewport is scrolled to the bottom.
func (m Model) AtBottom() bool { return m.yOffset >= m.maxYOffset() }

// ScrollPercent returns how far scrolled through the content [0,1] is, or 1
// if all content already fits (nothing to scroll).
func (m Model) ScrollPercent() float64 {
	if m.maxYOffset() <= 0 {
		return 1
	}
	return float64(m.yOffset) / float64(m.maxYOffset())
}

// Update handles arrow/page/home-end keys and mouse wheel scrolling.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	m.sync(false)
	switch msg := msg.(type) {
	case tui.Key:
		km := m.keys()
		next, prev := km.NextMatch, km.PrevMatch
		if len(next.Keys) == 0 && len(prev.Keys) == 0 {
			next, prev = defaultNextMatch(), defaultPrevMatch()
		}
		switch {
		case m.query != "" && keymap.Matches(msg, next):
			m.NextMatch()
		case m.query != "" && keymap.Matches(msg, prev):
			m.PrevMatch()
		case keymap.Matches(msg, km.Up):
			m.LineUp(1)
		case keymap.Matches(msg, km.Down):
			m.LineDown(1)
		case keymap.Matches(msg, km.PageUp):
			m.PageUp()
		case keymap.Matches(msg, km.PageDown):
			m.PageDown()
		case keymap.Matches(msg, km.Top):
			m.GotoTop()
		case keymap.Matches(msg, km.Bottom):
			m.GotoBottom()
		}
	case tui.MouseEvent:
		if msg.Action == tui.MouseActionPress {
			switch msg.Button {
			case tui.MouseButtonWheelUp:
				m.LineUp(3)
			case tui.MouseButtonWheelDown:
				m.LineDown(3)
			}
		}
	}
	return m, nil
}

// View renders exactly Height rows: the visible slice of content, padded
// with blank rows if there's less content than Height, each truncated to
// Width without breaking embedded ansi.Style codes.
func (m Model) View() string {
	if m.Height <= 0 {
		return ""
	}

	var vr []vrow
	n := m.LineCount()
	if m.wrapOn() {
		vr = m.vrows()
		n = len(vr)
	}

	rows := make([]string, m.Height)
	for i := 0; i < m.Height; i++ {
		if y := m.yOffset + i; y < n {
			line := m.paint(vr, y)
			if m.xOffset > 0 {
				line = ansi.TrimLeftWidth(line, m.xOffset)
			}
			rows[i] = ansi.Truncate(line, m.Width)
		}
	}
	return strings.Join(rows, "\n")
}

// DrawCells draws the viewport into r of buf, producing the screen View shows:
// the visible lines, scrolled by the offsets, clipped to Width x Height (and
// to r). It implements tui.CellDrawer and writes each line straight into the
// grid, without building a View string. Rows below the content are left as
// buf has them (blank on a fresh frame).
func (m Model) DrawCells(buf *cellbuf.Buffer, r cellbuf.Rect) {
	if m.Height <= 0 || m.Width <= 0 {
		return
	}
	r.W, r.H = min(r.W, m.Width), min(r.H, m.Height)
	sub := buf.Sub(r)
	var vr []vrow
	n := m.LineCount()
	if m.wrapOn() {
		vr = m.vrows()
		n = len(vr)
	}
	for i := 0; i < sub.Height(); i++ {
		y := m.yOffset + i
		if y >= n {
			break
		}
		var line string
		if m.wrapOn() || m.query != "" {
			line = m.paint(vr, y)
		} else {
			line = m.lines[m.head+y]
		}
		drawLine(sub, i, line, m.xOffset)
	}
}

// drawLine writes one content line on row y of sub, dropping its first xOff
// columns. A line the grid cannot represent is drawn as plain text.
func drawLine(sub *cellbuf.Buffer, y int, line string, xOff int) {
	if xOff > 0 {
		line = ansi.TrimLeftWidth(line, xOff)
	}
	if line == "" {
		return
	}
	if _, err := sub.SetStyled(0, y, line); err != nil {
		sub.SetString(0, y, line, 0)
	}
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// Compile-time proof that Model satisfies tui.Component[Model] — see
// tui.Component's doc comment for what this contract means and why.
var _ tui.Component[Model] = Model{}
